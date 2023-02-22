// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package events

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/jitter"
)

type ProcessPathData struct {
	Path string
	Args string
}

type ProcessPathInfo struct {
	Pid int
	ProcessPathData
}

type ProcessPathEntry struct {
	ProcessPathInfo
	expiresAt  time.Time
	fromKprobe bool
}

var numbersRegex = regexp.MustCompile(`\d+`)

// BPFProcessPathCache caches process path and args read via kprobes/proc
type BPFProcessPathCache struct {
	// Read-Write mutex for process path info
	lock sync.Mutex
	// Map of PID to process path information
	cache map[int]ProcessPathEntry

	// Ticker for running the GC thread that reaps expired entries.
	expireTicker jitter.JitterTicker
	// Max time for which an entry is retained.
	entryTTL time.Duration

	stopOnce         sync.Once
	wg               sync.WaitGroup
	stopC            chan struct{}
	eventProcessPath <-chan ProcessPath
}

// NewBPFProcessPathCache returns a new BPFProcessPathCache
func NewBPFProcessPathCache(eventProcessPathChan <-chan ProcessPath, gcInterval time.Duration,
	entryTTL time.Duration) *BPFProcessPathCache {
	return &BPFProcessPathCache{
		stopC:            make(chan struct{}),
		eventProcessPath: eventProcessPathChan,
		expireTicker:     jitter.NewTicker(gcInterval, gcInterval/10),
		entryTTL:         entryTTL,
		cache:            make(map[int]ProcessPathEntry),
		lock:             sync.Mutex{},
	}
}

func (r *BPFProcessPathCache) Start() error {
	log.Debugf("starting BPFProcessPathCache")
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		if r.eventProcessPath != nil {
			r.run()
		}
	}()

	return nil
}

func (r *BPFProcessPathCache) run() {
	defer r.expireTicker.Stop()
	for {
		select {
		case <-r.stopC:
			return
		case processEvent, ok := <-r.eventProcessPath:
			log.Debugf("Received process path event")
			if ok {
				info := convertPathEventToProcessPath(processEvent)
				log.Debugf("Converted event %+v to process path %+v", processEvent, info)
				r.updateCacheWithProcessPathInfo(info, true)
			}
		case <-r.expireTicker.Channel():
			r.expireCacheEntries()
		}
	}
}

func (r *BPFProcessPathCache) Stop() {
	r.stopOnce.Do(func() {
		close(r.stopC)
	})
	r.wg.Wait()
}

// Whenever process information is updated in the processInfoCache,
// process path cache is referred to see if there is path information.
// Process path information is updated either via the kprobes or
// by reading the /proc/pid/cmdline
func (r *BPFProcessPathCache) Lookup(Pid int) (ProcessPathInfo, bool) {
	r.lock.Lock()

	if entry, ok := r.cache[Pid]; ok {
		// Though the data is available from kprobes, we still do a check in /proc.
		// This is to avoid inconsistencies especially in cases like nginx deployments
		// where the kprobe data is that of the container process and proc data is
		// that of nginx. Hence if /proc/pid/cmdline is available that takes the higher
		// precedence.
		if entry.fromKprobe {
			procPath := fmt.Sprintf("/proc/%d/cmdline", Pid)
			_, err := os.Stat(procPath)
			if err == nil {
				log.Debugf("Process path found from kprobe. Reading /proc/%+v/cmdline", Pid)
				r.lock.Unlock()
				return r.getPathFromProc(Pid)
			}
		}
		log.Debugf("Found process path %+v for Pid %+v", entry.ProcessPathInfo, Pid)
		entry.expiresAt = time.Now().Add(r.entryTTL)
		r.cache[Pid] = entry
		r.lock.Unlock()
		return entry.ProcessPathInfo, true
	}
	r.lock.Unlock()
	log.Debugf("Process path not found in cache, reading /proc/%+v/cmdline", Pid)
	return r.getPathFromProc(Pid)
}

func (r *BPFProcessPathCache) getPathFromProc(Pid int) (ProcessPathInfo, bool) {
	// Read data from /proc/pid/cmdline
	// Add to processPathCache
	path, args, err := r.readProcCmdline(Pid)
	if err != nil {
		log.WithError(err).Debug("error reading /proc dir")
	} else {
		return ProcessPathInfo{
			Pid: Pid,
			ProcessPathData: ProcessPathData{
				Path: path,
				Args: args,
			},
		}, true
	}
	log.Debugf("Process path not found for PID %+v", Pid)
	return ProcessPathInfo{}, false
}

func (r *BPFProcessPathCache) updateCacheWithProcessPathInfo(info ProcessPathInfo, fromKprobe bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	log.Debugf("Updating process path info %+v", info)
	pid := info.Pid
	entry, ok := r.cache[pid]
	if ok {
		entry.ProcessPathData = info.ProcessPathData
		entry.expiresAt = time.Now().Add(r.entryTTL)
		log.Debugf("Process path cache updated with process data %+v", entry)
		r.cache[pid] = entry
	} else {
		entry := ProcessPathEntry{
			ProcessPathInfo: info,
			expiresAt:       time.Now().Add(r.entryTTL),
		}
		r.cache[pid] = entry
	}
	return
}

func (r *BPFProcessPathCache) expireCacheEntries() {
	r.lock.Lock()
	defer r.lock.Unlock()

	for pid, entry := range r.cache {
		if time.Until(entry.expiresAt) <= 0 {
			log.Debugf("Expiring process path %+v. Time until expiration %v", entry, time.Until(entry.expiresAt))
			delete(r.cache, pid)
			continue
		}
	}
}

func convertPathEventToProcessPath(event ProcessPath) ProcessPathInfo {
	return ProcessPathInfo{
		Pid: event.Pid,
		ProcessPathData: ProcessPathData{
			Path: event.Filename,
			Args: event.Arguments,
		},
	}
}

// readProcCmdline reads /proc/pid/cmdline to get the process path and the arguments.
// For this feature, hostPID will be set to true, thus enabling access to the complete
// /proc/pid inside the host
func (r *BPFProcessPathCache) readProcCmdline(procId int) (string, string, error) {
	var rpath, rargs, args string
	var rerror error
	IsPidPresent := false
	proc, err := os.ReadDir("/proc")
	if err != nil {
		return "", "", err
	}
	for _, f := range proc {
		if !f.IsDir() || !numbersRegex.MatchString(f.Name()) {
			continue
		}
		pid, err := strconv.Atoi(f.Name())
		if err != nil {
			log.Debugf("pid directory not numeric, skipping %+v", err)
			continue
		}
		procPath := fmt.Sprintf("/proc/%d/cmdline", pid)
		content, err := os.ReadFile(procPath)
		if err != nil {
			log.WithError(err).Debugf("error reading %v", procPath)
			if pid == procId {
				rerror = err
				IsPidPresent = true
			}
		} else {
			str := string(content)
			if len(str) == 0 {
				continue
			}
			pathArgs := strings.SplitN(strings.Replace(str, "\x00", " ", -1), " ", 2)
			path := pathArgs[0]
			if len(pathArgs) == 2 {
				args = pathArgs[1]
			}
			if pid == procId {
				rpath = path
				rargs = args
				IsPidPresent = true
			}
			pathInfo := ProcessPathInfo{
				Pid: pid,
				ProcessPathData: ProcessPathData{
					Path: path,
					Args: args,
				},
			}
			r.updateCacheWithProcessPathInfo(pathInfo, false)
		}
	}
	if !IsPidPresent {
		rerror = fmt.Errorf("pid %v directory not found in /proc", procId)
	}
	return rpath, rargs, rerror
}

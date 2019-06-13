package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/felixsyncer"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetLevel(log.WarnLevel)

	datastoreConfig := apiconfig.NewCalicoAPIConfig()
	datastoreConfig.Spec.DatastoreType = "etcdv3"
	datastoreConfig.Spec.EtcdEndpoints = "http://127.0.0.1:2379"
	backendClient, err := backend.NewClient(*datastoreConfig)
	if err != nil {
		panic(err)
	}

	syncer := felixsyncer.New(backendClient, datastoreConfig.Spec, new(SyncerCallbacksLogger))
	syncer.Start()

	log.Println(http.ListenAndServe("localhost:6060", nil))

	waitForExit()
}

func waitForExit() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println(sig)
		done <- true
	}()
	<-done
}

type SyncerCallbacksLogger struct {
}

func (a *SyncerCallbacksLogger) OnStatusUpdated(status api.SyncStatus) {
	log.Warnf("Status: %v", status)
	log.Warnf("Got %v items", len(allUpdates))
}

var allUpdates = make(map[string]bool)

func (a *SyncerCallbacksLogger) OnUpdates(updates []api.Update) {
	// CHANGE THIS CODE TO DO WHAT YOU WANT. COMMENTED OUT LINES CAN BE USED AS A GUIDE

	log.Warnf("Got %d updates\n", len(updates))
	for _, update := range updates {
		fmt.Printf("Update: %v\n", update)
		switch update.Key.(type) {
		default:
		case model.HostEndpointKey, model.WorkloadEndpointKey, model.ProfileRulesKey:
			//fmt.Printf("HEP Profile: %v\n", update.Value.(*model.HostEndpoint).ProfileIDs)
			//case model.WorkloadEndpointKey:
			//	//fmt.Printf("WEP Profile: %v\n", update.Value.(*model.WorkloadEndpoint).ProfileIDs)
			//case model.ProfileRulesKey:
			//	//spew.Dump(update)
			//	//fmt.Printf("Profile: %v=%v\n", update.Key, update.Value.(*model.ProfileRules))
			switch update.UpdateType {
			case api.UpdateTypeKVNew:
				//fmt.Println(update.Key.String())
				allUpdates[update.Key.String()] = true
			case api.UpdateTypeKVDeleted:
				//fmt.Println(update.Key.String())
				delete(allUpdates, update.Key.String())
			default:
				panic("Unknown")
			}
			if len(allUpdates)%1000 == 0 {
				log.Warnf("Got %v items", len(allUpdates))
				log.Warnf("Goroutines: %v", runtime.NumGoroutine())
			}
		}

		//if update.UpdateType == api.UpdateTypeKVNew {
	}
}

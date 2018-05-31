// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/containernetworking/cni/pkg/ns"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
)

type podSpec struct {
	// Actually there's no way to specify a pod's MAC address through the Kubernetes datastore.
	// It's only used for an optimization anyway (to save an ARP request when routing on the
	// last hop to a pod), so probably acceptable that we don't have it.
	mac string
	// IPv4 address.  If not provided, createPod will generate one.
	ipv4Addr string
	// IPv6 address.  Not yet supported by Kubernetes.
	ipv6Addr string
	// Pod name.  If not provided, createPod will generate one.
	name string
	// Labels (may be nil).
	labels map[string]string
}

type localNetworking struct {
	podIf     netlink.Link
	hostIf    netlink.Link
	namespace ns.NetNS
}

var localNetworkingMap = map[string]*localNetworking{}
var localNetworkingMutex = sync.Mutex{}

func createPod(clientset *kubernetes.Clientset, d deployment, nsName string, spec podSpec) *v1.Pod {
	// Create a handle for our operations in this function, this ensures that they all go through the
	// same netlink socket.  Doing that seems to work around some consistency issues, where we would create
	// the link but then LinkByName wouldn't find it.  It's not clear why doing that helps but it
	// may be that the kernel enforces consistency when you re-use the same socket, or, it may be
	// that load causes the issue and we put less load on the kernel.
	handle, err := netlink.NewHandle()
	panicIfError(err)
	defer handle.Delete()

	name := spec.name
	if name == "" {
		name = fmt.Sprintf("run%d", rand.Uint32())
	}
	host := d.ChooseHost(clientset)
	ip := spec.ipv4Addr
	if ip == "" {
		ip = GetNextPodAddr()
	}
	pod_in := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1.PodSpec{Containers: []v1.Container{{
			Name:  fmt.Sprintf("container-%s", name),
			Image: "ignore",
		}},
			NodeName: host.name,
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			Conditions: []v1.PodCondition{{
				Type:   v1.PodScheduled,
				Status: v1.ConditionTrue,
			}},
			PodIP: ip,
		},
	}
	if spec.labels != nil {
		pod_in.ObjectMeta.Labels = spec.labels
	}
	log.WithField("pod_in", pod_in).Debug("Pod defined")
	pod_out, err := clientset.CoreV1().Pods(nsName).Create(pod_in)
	if err != nil {
		panic(err)
	}
	log.WithField("pod_out", pod_out).Debug("Created pod")
	pod_in = pod_out
	pod_in.Status.PodIP = ip
	pod_out, err = clientset.CoreV1().Pods(nsName).UpdateStatus(pod_in)
	if err != nil {
		panic(err)
	}
	log.WithField("pod_out", pod_out).Debug("Updated pod")

	if host.isLocal {
		// Create the cali interface, so that Felix does dataplane programming for the local
		// endpoint.
		interfaceName := conversion.VethNameForWorkload(nsName, name)
		log.WithField("interfaceName", interfaceName).Info("Prepare interface")

		// Create a namespace.
		podNamespace, err := ns.NewNS()
		panicIfError(err)
		log.WithField("podNamespace", podNamespace).Debug("Created namespace")

		// Create a veth pair.
		veth := &netlink.Veth{
			LinkAttrs: netlink.LinkAttrs{Name: interfaceName},
			PeerName:  "p" + interfaceName[1:],
		}

		err = handle.LinkAdd(veth)
		panicIfError(err)
		log.WithField("veth", veth).Debug("Created veth pair")

		// Move the pod end of the pair into the namespace, and set it up.
		linkByNameRetries := 3
	retry:
		podIf, err := handle.LinkByName(veth.PeerName)
		log.WithField("podIf", podIf).Debug("Pod end")
		if (err != nil) && linkByNameRetries > 0 {
			log.WithField("name", veth.PeerName).WithError(err).Info("LinkByName failed, retrying...")
			linkByNameRetries--
			time.Sleep(500 * time.Millisecond)
			goto retry
		}
		panicIfError(err)
		err = handle.LinkSetNsFd(podIf, int(podNamespace.Fd()))
		panicIfError(err)

		err = podNamespace.Do(func(_ ns.NetNS) (err error) {
			err = runCommand("ip", "link", "set", veth.PeerName, "up")
			if err != nil {
				return
			}
			err = runCommand("ip", "addr", "add", ip+"/32", "dev", veth.PeerName)
			if err != nil {
				return
			}
			err = runCommand("ip", "route", "add", "169.254.169.254/32", "dev", veth.PeerName)
			if err != nil {
				return
			}
			err = runCommand("ip", "route", "add", "default", "via", "169.254.169.254", "dev", veth.PeerName)
			return
		})
		panicIfError(err)

		// Set the host end up too.
		hostIf, err := handle.LinkByName(veth.LinkAttrs.Name)
		log.WithField("hostIf", hostIf).Debug("Host end")
		panicIfError(err)
		err = handle.LinkSetUp(hostIf)
		panicIfError(err)

		// Lock mutex, to enable pod creation from multiple goroutines.
		localNetworkingMutex.Lock()
		defer localNetworkingMutex.Unlock()

		localNetworkingMap[nsName+"."+name] = &localNetworking{
			podIf:     podIf,
			hostIf:    hostIf,
			namespace: podNamespace,
		}
	}
	return pod_out
}

func removeLocalPodNetworking(pod *v1.Pod) {
	// Retrieve local networking details for this pod.
	key := pod.ObjectMeta.Namespace + "." + pod.ObjectMeta.Name

	// Lock mutex, as we do pod cleanup from multiple goroutines.
	localNetworkingMutex.Lock()
	defer localNetworkingMutex.Unlock()

	ln := localNetworkingMap[key]
	if ln != nil {
		log.WithField("key", key).Info("Cleanup local networking")

		// Delete pod-side interface.
		err := ln.namespace.Do(func(_ ns.NetNS) error {
			return netlink.LinkDel(ln.podIf)
		})
		panicIfError(err)

		// Delete host-side interface.  Actually it seems this has already happened as part
		// of the pod-side interface being deleted just above; so we don't need a separate
		// operation here.
		//err = netlink.LinkDel(ln.hostIf)
		//panicIfError(err)

		// Delete namespace.
		err = ln.namespace.Close()
		panicIfError(err)

		// Delete local networking details.
		delete(localNetworkingMap, key)
	}

	return
}

var GetNextPodAddr = ipAddrAllocator("10.28.%d.%d")

func cleanupAllPods(clientset *kubernetes.Clientset, nsPrefix string) {
	log.WithField("nsPrefix", nsPrefix).Info("Cleaning up all pods...")
	nsList, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	log.WithField("count", len(nsList.Items)).Info("Namespaces present")
	podsDeleted := 0
	admission := make(chan int, 10)
	waiter := sync.WaitGroup{}
	waiter.Add(len(nsList.Items))
	for _, ns := range nsList.Items {
		nsName := ns.ObjectMeta.Name
		go func() {
			admission <- 1
			if strings.HasPrefix(nsName, nsPrefix) {
				podList, err := clientset.CoreV1().Pods(nsName).List(metav1.ListOptions{})
				if err != nil {
					panic(err)
				}
				log.WithField("count", len(podList.Items)).WithField("namespace", nsName).Debug(
					"Pods present")
				for _, pod := range podList.Items {
					err = clientset.CoreV1().Pods(nsName).Delete(pod.ObjectMeta.Name, deleteImmediately)
					if err != nil {
						panic(err)
					}
					removeLocalPodNetworking(&pod)
				}
				podsDeleted += len(podList.Items)
			}
			<-admission
			waiter.Done()
		}()
	}
	waiter.Wait()

	Eventually(getNumEndpointsDefault(-1), "30s", "1s").Should(
		BeNumerically("==", 0),
		"Removal of pods wasn't reflected in Felix metrics",
	)
	log.WithField("podsDeleted", podsDeleted).Info("Cleaned up all pods")
}

var zeroGracePeriod int64 = 0

var deleteImmediately = &metav1.DeleteOptions{GracePeriodSeconds: &zeroGracePeriod}

func runCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	log.Infof("Running '%s %s'", cmd.Path, strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	log.WithField("rc", err).Infof("output: %v", string(output))
	return err
}

func runNmap(pod1, pod2 *v1.Pod) {
	log.WithField("from", pod1.Status.PodIP).WithField("to", pod2.Status.PodIP).Info("Run nmap")

	// Lock mutex, as we do pod cleanup from multiple goroutines.
	localNetworkingMutex.Lock()
	defer localNetworkingMutex.Unlock()

	err := localNetworkingMap[pod1.ObjectMeta.Namespace+"."+pod1.ObjectMeta.Name].namespace.Do(func(_ ns.NetNS) (err error) {
		runCommand("ping", "-c", "1", "-W", "1", pod2.Status.PodIP)
		runCommand("nmap", "-Pn", "-T5", pod2.Status.PodIP)
		return
	})
	panicIfError(err)
}

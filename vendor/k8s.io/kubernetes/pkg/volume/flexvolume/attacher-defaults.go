/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package flexvolume

import (
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/volume"
)

type attacherDefaults flexVolumeAttacher

// Attach is part of the volume.Attacher interface
func (a *attacherDefaults) Attach(spec *volume.Spec, hostName types.NodeName) (string, error) {
	glog.Warning(logPrefix(a.plugin.flexVolumePlugin), "using default Attach for volume ", spec.Name, ", host ", hostName)
	return "", nil
}

// WaitForAttach is part of the volume.Attacher interface
func (a *attacherDefaults) WaitForAttach(spec *volume.Spec, devicePath string, timeout time.Duration) (string, error) {
	glog.Warning(logPrefix(a.plugin.flexVolumePlugin), "using default WaitForAttach for volume ", spec.Name, ", device ", devicePath)
	return devicePath, nil
}

// GetDeviceMountPath is part of the volume.Attacher interface
func (a *attacherDefaults) GetDeviceMountPath(spec *volume.Spec, mountsDir string) (string, error) {
	return a.plugin.getDeviceMountPath(spec)
}

// MountDevice is part of the volume.Attacher interface
func (a *attacherDefaults) MountDevice(spec *volume.Spec, devicePath string, deviceMountPath string, mounter mount.Interface) error {
	glog.Warning(logPrefix(a.plugin.flexVolumePlugin), "using default MountDevice for volume ", spec.Name, ", device ", devicePath, ", deviceMountPath ", deviceMountPath)
	volSource, readOnly := getVolumeSource(spec)

	options := make([]string, 0)

	if readOnly {
		options = append(options, "ro")
	} else {
		options = append(options, "rw")
	}

	diskMounter := &mount.SafeFormatAndMount{Interface: mounter, Exec: a.plugin.host.GetExec(a.plugin.GetPluginName())}

	return diskMounter.FormatAndMount(devicePath, deviceMountPath, volSource.FSType, options)
}

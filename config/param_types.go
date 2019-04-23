// Copyright (c) 2016-2019 Tigera, Inc. All rights reserved.
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

package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"time"

	"github.com/kardianos/osext"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cnet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
)

const (
	MinIptablesMarkBits = 2
)

type Metadata struct {
	Name              string
	Default           interface{}
	ZeroValue         interface{}
	NonZero           bool
	DieOnParseFailure bool
	Local             bool
}

func (m *Metadata) GetMetadata() *Metadata {
	return m
}

func (m *Metadata) parseFailed(raw, msg string) error {
	return errors.New(
		fmt.Sprintf("Failed to parse config parameter %v; value %#v: %v",
			m.Name, raw, msg))
}

func (m *Metadata) setDefault(config *Config) {
	log.Debugf("Defaulting: %v to %v", m.Name, m.Default)
	field := reflect.ValueOf(config).Elem().FieldByName(m.Name)
	value := reflect.ValueOf(m.Default)
	field.Set(value)
}

type BoolParam struct {
	Metadata
}

func (p *BoolParam) Parse(raw string) (interface{}, error) {
	switch strings.ToLower(raw) {
	case "true", "1", "yes", "y", "t":
		return true, nil
	case "false", "0", "no", "n", "f":
		return false, nil
	}
	return nil, p.parseFailed(raw, "invalid boolean")
}

type MinMax struct {
	Min int
	Max int
}

type IntParam struct {
	Metadata
	Ranges []MinMax
}

func (p *IntParam) Parse(raw string) (interface{}, error) {
	value, err := strconv.ParseInt(raw, 0, 64)
	if err != nil {
		err = p.parseFailed(raw, "invalid int")
		return nil, err
	}
	result := int(value)
	if len(p.Ranges) == 1 {
		if result < p.Ranges[0].Min {
			err = p.parseFailed(raw,
				fmt.Sprintf("value must be at least %v", p.Ranges[0].Min))
		} else if result > p.Ranges[0].Max {
			err = p.parseFailed(raw,
				fmt.Sprintf("value must be at most %v", p.Ranges[0].Max))
		}
	} else {
		good := false
		for _, r := range p.Ranges {
			if result >= r.Min && result <= r.Max {
				good = true
				break
			}
		}
		if !good {
			msg := "value must be one of"
			for _, r := range p.Ranges {
				if r.Min == r.Max {
					msg = msg + fmt.Sprintf(" %v", r.Min)
				} else {
					msg = msg + fmt.Sprintf(" %v-%v", r.Min, r.Max)
				}
			}
			err = p.parseFailed(raw, msg)
		}
	}
	return result, err
}

type Int32Param struct {
	Metadata
}

func (p *Int32Param) Parse(raw string) (interface{}, error) {
	value, err := strconv.ParseInt(raw, 0, 32)
	if err != nil {
		err = p.parseFailed(raw, "invalid 32-bit int")
		return nil, err
	}
	result := int32(value)
	return result, err
}

type FloatParam struct {
	Metadata
}

func (p *FloatParam) Parse(raw string) (result interface{}, err error) {
	result, err = strconv.ParseFloat(raw, 64)
	if err != nil {
		err = p.parseFailed(raw, "invalid float")
		return
	}
	return
}

type SecondsParam struct {
	Metadata
	Min int
	Max int
}

func (p *SecondsParam) Parse(raw string) (result interface{}, err error) {
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		err = p.parseFailed(raw, "invalid float")
		return
	}
	result = time.Duration(seconds * float64(time.Second))
	if int(seconds) < p.Min {
		err = p.parseFailed(raw, fmt.Sprintf("value must be at least %v", p.Min))
	} else if int(seconds) > p.Max {
		err = p.parseFailed(raw, fmt.Sprintf("value must be at most %v", p.Max))
	}
	return result, err
	return
}

type MillisParam struct {
	Metadata
}

func (p *MillisParam) Parse(raw string) (result interface{}, err error) {
	millis, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		err = p.parseFailed(raw, "invalid float")
		return
	}
	result = time.Duration(millis * float64(time.Millisecond))
	return
}

type RegexpParam struct {
	Metadata
	Regexp *regexp.Regexp
	Msg    string
}

func (p *RegexpParam) Parse(raw string) (result interface{}, err error) {
	if !p.Regexp.MatchString(raw) {
		err = p.parseFailed(raw, p.Msg)
	} else {
		result = raw
	}
	return
}

type FileParam struct {
	Metadata
	MustExist  bool
	Executable bool
}

func (p *FileParam) Parse(raw string) (interface{}, error) {
	if p.Executable {
		// Special case: for executable files, we search our directory
		// and the system path.
		logCxt := log.WithField("name", raw)
		var path string
		if myDir, err := osext.ExecutableFolder(); err == nil {
			logCxt.WithField("myDir", myDir).Info(
				"Looking for executable in my directory")
			path = myDir + string(os.PathSeparator) + raw
			stat, err := os.Stat(path)
			if err == nil {
				if m := stat.Mode(); !m.IsDir() && m&0111 > 0 {
					return path, nil
				}
			} else {
				logCxt.WithField("myDir", myDir).Info(
					"No executable in my directory")
				path = ""
			}
		} else {
			logCxt.WithError(err).Warn("Failed to get my dir")
		}
		if path == "" {
			logCxt.Info("Looking for executable on path")
			var err error
			path, err = exec.LookPath(raw)
			if err != nil {
				logCxt.WithError(err).Warn("Path lookup failed")
				path = ""
			}
		}
		if path == "" && p.MustExist {
			log.Error("Executable missing")
			return nil, p.parseFailed(raw, "missing file")
		}
		log.WithField("path", path).Info("Executable path")
		return path, nil
	} else if p.MustExist && raw != "" {
		log.WithField("path", raw).Info("Looking for required file")
		_, err := os.Stat(raw)
		if err != nil {
			log.Errorf("Failed to access %v: %v", raw, err)
			return nil, p.parseFailed(raw, "failed to access file")
		}
	}
	return raw, nil
}

type Ipv4Param struct {
	Metadata
}

func (p *Ipv4Param) Parse(raw string) (result interface{}, err error) {
	result = net.ParseIP(raw)
	if result == nil {
		err = p.parseFailed(raw, "invalid IP")
	}
	return
}

type PortListParam struct {
	Metadata
}

func (p *PortListParam) Parse(raw string) (interface{}, error) {
	var result []ProtoPort
	for _, portStr := range strings.Split(raw, ",") {
		portStr = strings.Trim(portStr, " ")
		if portStr == "" {
			continue
		}

		parts := strings.Split(portStr, ":")
		if len(parts) > 2 {
			return nil, p.parseFailed(raw,
				"ports should be <protocol>:<number> or <number>")
		}
		protocolStr := "tcp"
		if len(parts) > 1 {
			protocolStr = strings.ToLower(parts[0])
			portStr = parts[1]
		}
		if protocolStr != "tcp" && protocolStr != "udp" {
			return nil, p.parseFailed(raw, "unknown protocol: "+protocolStr)
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			err = p.parseFailed(raw, "ports should be integers")
			return nil, err
		}
		if port < 0 || port > 65535 {
			err = p.parseFailed(raw, "ports must be in range 0-65535")
			return nil, err
		}
		result = append(result, ProtoPort{
			Protocol: protocolStr,
			Port:     uint16(port),
		})
	}
	return result, nil
}

type PortRangeParam struct {
	Metadata
}

func (p *PortRangeParam) Parse(raw string) (interface{}, error) {
	portRange, err := numorstring.PortFromString(raw)
	if err != nil {
		return nil, p.parseFailed(raw, fmt.Sprintf("%s is not a valid port range", raw))
	}
	if len(portRange.PortName) > 0 {
		return nil, p.parseFailed(raw, fmt.Sprintf("%s has port name set", raw))
	}
	return portRange, nil
}

type PortRangeListParam struct {
	Metadata
}

func (p *PortRangeListParam) Parse(raw string) (interface{}, error) {
	var result []numorstring.Port
	for _, rangeStr := range strings.Split(raw, ",") {
		portRange, err := numorstring.PortFromString(rangeStr)
		if err != nil {
			return nil, p.parseFailed(raw, fmt.Sprintf("%s is not a valid port range", rangeStr))
		}
		if len(portRange.PortName) > 0 {
			return nil, p.parseFailed(raw, fmt.Sprintf("%s has port name set", rangeStr))
		}
		result = append(result, portRange)
	}
	return result, nil
}

type EndpointListParam struct {
	Metadata
}

func (p *EndpointListParam) Parse(raw string) (result interface{}, err error) {
	value := strings.Split(raw, ",")
	scheme := ""
	resultSlice := []string{}
	for _, endpoint := range value {
		endpoint = strings.Trim(endpoint, " ")
		if len(endpoint) == 0 {
			continue
		}
		var u *url.URL
		u, err = url.Parse(endpoint)
		if err != nil {
			err = p.parseFailed(raw,
				fmt.Sprintf("%v is not a valid URL", endpoint))
			return
		}
		if scheme != "" && u.Scheme != scheme {
			err = p.parseFailed(raw,
				"all endpoints must have the same scheme")
			return
		}
		if u.Path == "" {
			u.Path = "/"
		}
		if u.Opaque != "" || u.User != nil || u.Path != "/" ||
			u.RawPath != "" || u.RawQuery != "" ||
			u.Fragment != "" {
			log.WithField("url", fmt.Sprintf("%#v", u)).Error(
				"Unsupported URL part")
			err = p.parseFailed(raw,
				"endpoint contained unsupported URL part; "+
					"expected http(s)://hostname:port only.")
			return
		}
		resultSlice = append(resultSlice, u.String())
	}
	result = resultSlice
	return
}

type MarkBitmaskParam struct {
	Metadata
}

func (p *MarkBitmaskParam) Parse(raw string) (interface{}, error) {
	value, err := strconv.ParseUint(raw, 0, 32)
	if err != nil {
		log.Warningf("Failed to parse %#v as an int: %v", raw, err)
		err = p.parseFailed(raw, "invalid mark: should be 32-bit int")
		return nil, err
	}
	result := uint32(value)
	bitCount := uint32(0)
	for i := uint(0); i < 32; i++ {
		bit := (result >> i) & 1
		bitCount += bit
	}
	if bitCount < MinIptablesMarkBits {
		err = p.parseFailed(raw,
			fmt.Sprintf("invalid mark: needs to have %v bits set",
				MinIptablesMarkBits))
	}
	return result, err
}

type OneofListParam struct {
	Metadata
	lowerCaseOptionsToCanonical map[string]string
}

func (p *OneofListParam) Parse(raw string) (result interface{}, err error) {
	result, ok := p.lowerCaseOptionsToCanonical[strings.ToLower(raw)]
	if !ok {
		err = p.parseFailed(raw, "unknown option")
	}
	return
}

type CIDRListParam struct {
	Metadata
}

func (c *CIDRListParam) Parse(raw string) (result interface{}, err error) {
	log.WithField("CIDRs raw", raw).Info("CIDRList")
	values := strings.Split(raw, ",")
	resultSlice := []string{}
	for _, in := range values {
		val := strings.Trim(in, " ")
		if len(val) == 0 {
			continue
		}
		ip, net, e := cnet.ParseCIDROrIP(val)
		if e != nil {
			err = c.parseFailed(in, "invalid CIDR or IP "+val)
			return
		}
		if ip.Version() != 4 {
			err = c.parseFailed(in, "invalid CIDR or IP (not v4)")
			return
		}
		resultSlice = append(resultSlice, net.String())
	}
	return resultSlice, nil
}

type ServerListParam struct {
	Metadata
}

const k8sServicePrefix = "k8s-service:"

func (c *ServerListParam) Parse(raw string) (result interface{}, err error) {
	log.WithField("raw", raw).Info("ServerList")
	values := strings.Split(raw, ",")
	resultSlice := []string{}
	for _, in := range values {
		val := strings.TrimSpace(in)
		if len(val) == 0 {
			continue
		}
		if strings.HasPrefix(val, k8sServicePrefix) {
			svcName := val[len(k8sServicePrefix):]
			svc, e := GetKubernetesService("kube-system", svcName)
			if e != nil {
				// Warn but don't report parse failure, so that other trusted IPs
				// can still take effect.
				log.Warningf("Couldn't get Kubernetes service '%v': %v", svcName, e)
				continue
			}
			val = svc.Spec.ClusterIP
			if val == "" {
				// Ditto.
				log.Warningf("Kubernetes service '%v' has no ClusterIP", svcName)
				continue
			}
		} else if net.ParseIP(val) == nil {
			err = c.parseFailed(in, "invalid server specification '"+val+"'")
			return
		}
		resultSlice = append(resultSlice, val)
	}
	return resultSlice, nil
}

func GetKubernetesService(namespace, svcName string) (*v1.Service, error) {
	k8sconf, err := rest.InClusterConfig()
	if err != nil {
		log.WithError(err).Info("Unable to create Kubernetes config.")
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(k8sconf)
	if err != nil {
		log.WithError(err).Error("Unable to create Kubernetes client set.")
		return nil, err
	}
	svcClient := clientset.CoreV1().Services(namespace)
	return svcClient.Get(svcName, metav1.GetOptions{})
}

type RegionParam struct {
	Metadata
}

const regionNamespacePrefix = "openstack-region-"
const maxRegionLength int = validation.DNS1123LabelMaxLength - len(regionNamespacePrefix)

func (r *RegionParam) Parse(raw string) (result interface{}, err error) {
	log.WithField("raw", raw).Info("Region")
	if len(raw) > maxRegionLength {
		err = fmt.Errorf("The value of OpenstackRegion must be %v chars or fewer", maxRegionLength)
		return
	}
	errs := validation.IsDNS1123Label(raw)
	if len(errs) != 0 {
		msg := "The value of OpenstackRegion must be a valid DNS label"
		for _, err := range errs {
			msg = msg + "; " + err
		}
		err = errors.New(msg)
		return
	}
	return raw, nil
}

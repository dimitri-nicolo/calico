// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package sidecar

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"

	"github.com/projectcalico/calico/l7admissioncontroller/cmd/l7admctrl/config"
)

type sidecarWebhook struct {
	deserializer runtime.Decoder
}

func NewSidecarHandler() http.Handler {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(admissionv1.AddToScheme(scheme))
	utilruntime.Must(admissionregistrationv1.AddToScheme(scheme))

	return &sidecarWebhook{
		deserializer: codecs.UniversalDeserializer(),
	}
}

func (s *sidecarWebhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		res  runtime.Object
		obj  runtime.Object
		gvk  *schema.GroupVersionKind
		body []byte
		err  error
	)

	// Check content-type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.Errorf("contentType=%s, expect application/json", contentType)
		goto badRequest
	}

	// Parse body
	body, err = io.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("Request body could not be read: %v", err)
		goto badRequest
	}
	obj, gvk, err = s.deserializer.Decode(body, nil, nil)
	if err != nil {
		klog.Errorf("Request could not be decoded: %v", err)
		goto badRequest
	}
	switch *gvk {
	case v1.SchemeGroupVersion.WithKind("AdmissionReview"):
		admrev, ok := obj.(*v1.AdmissionReview)
		if !ok {
			klog.Errorf("Expected v1.AdmissionReview but got %T", obj)
			goto badRequest
		}
		resAdmrev := &v1.AdmissionReview{}
		resAdmrev.SetGroupVersionKind(*gvk)
		resAdmrev.Response = &v1.AdmissionResponse{
			UID:     admrev.Request.UID,
			Allowed: true,
		}
		err = s.patch(resAdmrev.Response, admrev.Request)
		if err != nil {
			klog.Error(err)
			goto internalErr
		}
		res = resAdmrev
	default:
		klog.Errorf("Unsupported group version kind: %v", gvk)
		goto badRequest
	}

	body, err = json.Marshal(res)
	if err != nil {
		klog.Error(err)
		goto internalErr
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(body); err != nil {
		klog.Error(err)
	}
	return

internalErr:
	w.WriteHeader(http.StatusInternalServerError)
	return

badRequest:
	w.WriteHeader(http.StatusBadRequest)
}

var (
	defaultVolumes = []string{
		`{"name":"envoy-config","emptyDir":{}}`,
		`{"name":"dikastes-sock","hostPath":{"path":"/var/run/dikastes","type":"Directory"}}`,
		`{"name":"l7-collector-sock","hostPath":{"path":"/var/run/l7-collector","type":"Directory"}}`,
	}
)

const (
	tmplDikastesInit = `{
		"name":"tigera-dikastes-init",
		"image":"%s",
		"command":["/dikastes","init"],
		"args":[%s],
		"env":[
			{"name":"DIKASTES_POD_NAMESPACE","valueFrom":{"fieldRef":{"fieldPath":"metadata.namespace"}}},
			{"name":"DIKASTES_POD_NAME","valueFrom":{"fieldRef":{"fieldPath":"metadata.name"}}}
		],
		"volumeMounts":[
			{"name":"envoy-config","mountPath":"/etc/tigera"}
		],
		"securityContext":{
			"capabilities":{
				"add":["NET_ADMIN","NET_RAW"]
			}
		}
	}`
	tmplEnvoy = `{
		"name":"tigera-envoy",
		"image":"%s",
		"command":["envoy","-c","/etc/tigera/envoy.yaml","-l","trace"],
		"restartPolicy":"Always",
		"ports":[{"containerPort":16001}],
		"startupProbe":{
			"tcpSocket":{
				"port":16001
			}
		},
		"volumeMounts":[
			{"name":"envoy-config","mountPath":"/etc/tigera"},
			{"name":"dikastes-sock","mountPath":"/var/run/dikastes"}
			{"name":"dikastes-sock","mountPath":"/var/run/l7-collector"}
		]
	}`
)

type sidecarCfg struct {
	dikastesImg  string
	envoyImg     string
	logging      bool
	policy       bool
	waf          bool
	wafConfigMap string
}

func (cfg *sidecarCfg) volumes() []string {
	volumes := append([]string(nil), defaultVolumes...)

	if cfg.logging || cfg.policy {
		volumes = append(volumes, `{"name":"felix-sync","csi":{"driver":"csi.tigera.io"}}`)
	}
	if cfg.waf {
		volumes = append(volumes, `{"name":"tigera-waf-logfiles","hostPath":{"path":"/var/log/calico/waf","type":"DirectoryOrCreate"}}`)
		if cfg.wafConfigMap != "" {
			volumes = append(volumes, `{"name":"tigera-waf-ruleset","configMap":{"defaultMode":420,"name":"`+cfg.wafConfigMap+`"}}`)
		}
	}

	return volumes
}

func (cfg *sidecarCfg) dikastesInitArgs() string {
	args := []string{}

	if cfg.logging {
		args = append(args, `"--logs-enabled"`)
	}
	if cfg.policy {
		args = append(args, `"--alp-enabled"`)
	}
	if cfg.waf {
		args = append(args, `"--waf-enabled"`)
	}

	return strings.Join(args, ",")
}

func (s *sidecarWebhook) patch(res *v1.AdmissionResponse, req *v1.AdmissionRequest) error {
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		return err
	}

	cfg := sidecarCfg{
		dikastesImg: config.DikastesImg,
		envoyImg:    config.EnvoyImg,
		logging:     (pod.ObjectMeta.Annotations["applicationlayer.projectcalico.org/logging"] == "Enabled"),
		policy:      (pod.ObjectMeta.Annotations["applicationlayer.projectcalico.org/policy"] == "Enabled"),
		waf:         (pod.ObjectMeta.Annotations["applicationlayer.projectcalico.org/waf"] == "Enabled"),
	}
	if cfg.waf {
		cfg.wafConfigMap = pod.ObjectMeta.Annotations["applicationlayer.projectcalico.org/wafConfigMap"]
	}

	if !(cfg.logging || cfg.policy || cfg.waf) {
		return nil
	}

	// injects volumes and initContainers
	volumes := []string{}
	for _, v := range cfg.volumes() {
		volumes = append(volumes, fmt.Sprintf(
			`{"op":"add","path":"/spec/volumes/-","value":%s}`,
			v,
		))
	}
	initContainers := fmt.Sprintf(
		`{"op":"add","path":"/spec/initContainers","value":%s}`,
		"["+strings.Join([]string{
			fmt.Sprintf(tmplDikastesInit, config.DikastesImg, cfg.dikastesInitArgs()),
			fmt.Sprintf(tmplEnvoy, config.EnvoyImg),
		}, ",")+"]",
	)

	pt := v1.PatchTypeJSONPatch
	res.PatchType = &pt
	res.Patch = []byte(fmt.Sprintf(`[%s]`,
		strings.Join(append(volumes, initContainers), ","),
	))

	return nil
}

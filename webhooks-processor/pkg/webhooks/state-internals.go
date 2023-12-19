// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cnf/structhash"
	"github.com/sirupsen/logrus"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/webhooks-processor/pkg/helpers"
)

const (
	ConfigVarNamespace      = "tigera-intrusion-detection"
	WebhookLabelsAnnotation = "webhooks.projectcalico.org/labels"
	ConditionHealthy        = "Healthy"
	ConditionHealthyDesc    = "the webhook is healthy"
	ConditionLastFetch      = "EventsFetched"
	ConditionLastFetchDesc  = ""
)

func (s *ControllerState) startNewInstance(ctx context.Context, webhook *api.SecurityEventWebhook) {
	logrus.WithField("uid", webhook.UID).Info("Webhook validation process started")

	if webhook.Spec.State == api.SecurityEventWebhookStateDisabled {
		s.preventRestarts[webhook.UID] = true
		s.updateWebhookHealth(webhook, "WebhookState", time.Now(), errors.New("the webhook has been disabled"))
		return
	}
	parsedQuery, err := query.ParseQuery(webhook.Spec.Query)
	if err != nil {
		s.preventRestarts[webhook.UID] = true
		s.updateWebhookHealth(webhook, "QueryParsing", time.Now(), err)
		return
	}
	err = query.Validate(parsedQuery, query.IsValidEventsKeysAtom)
	if err != nil {
		s.preventRestarts[webhook.UID] = true
		s.updateWebhookHealth(webhook, "QueryValidation", time.Now(), err)
		return
	}
	config, err := s.parseConfig(ctx, webhook.Spec.Config)
	if err != nil {
		s.preventRestarts[webhook.UID] = true
		s.updateWebhookHealth(webhook, "ConfigurationParsing", time.Now(), err)
		return
	}
	provider, ok := s.config.Providers[webhook.Spec.Consumer]
	if !ok {
		s.preventRestarts[webhook.UID] = true
		s.updateWebhookHealth(webhook, "ConsumerDiscovery", time.Now(), fmt.Errorf("unknown consumer: %s", webhook.Spec.Consumer))
		return
	}
	if err = provider.Validate(config); err != nil {
		s.preventRestarts[webhook.UID] = true
		s.updateWebhookHealth(webhook, "ConsumerConfigurationValidation", time.Now(), err)
		return
	}

	processFunc := provider.Process
	if webhook.Spec.State == api.SecurityEventWebhookStateDebug {
		processFunc = s.debugProcessFunc(webhook)
	}
	webhookCtx, cancelFunc := context.WithCancel(ctx)
	webhookUpdateChan := make(chan *api.SecurityEventWebhook)
	specHash := string(structhash.Md5(webhook.Spec, 1))

	s.webhooksTrail[webhook.UID] = &webhookState{
		specHash:       specHash,
		cancelFunc:     cancelFunc,
		webhookUpdates: webhookUpdateChan,
	}

	rateLimiter := helpers.NewRateLimiter(provider.Config().RateLimiterDuration, provider.Config().RateLimiterCount)

	s.wg.Add(1)
	go s.webhookGoroutine(webhookCtx, config, parsedQuery, processFunc, webhookUpdateChan, webhook, rateLimiter)
	s.updateWebhookHealth(webhook, "WebhookValidation", time.Now(), nil)

	logrus.WithField("uid", webhook.UID).Info("Webhook validated and registered")
}

func (s *ControllerState) parseConfig(ctx context.Context, config []api.SecurityEventWebhookConfigVar) (map[string]string, error) {
	parsed := make(map[string]string)
	for _, configItem := range config {
		if configItem.ValueFrom == nil {
			parsed[configItem.Name] = configItem.Value
			continue
		}
		value, err := s.retrieveConfigValue(ctx, configItem.ValueFrom)
		if err == nil {
			parsed[configItem.Name] = value
			continue
		}
		return nil, err
	}
	return parsed, nil
}

func (s *ControllerState) updateWebhookHealth(webhook *api.SecurityEventWebhook, reason string, timestamp time.Time, err error) {
	var status metav1.ConditionStatus
	var message string

	logEntry := logrus.WithFields(logrus.Fields{
		"webhook.Name": webhook.Name,
		"reason":       reason,
		"timestamp":    timestamp,
	}).WithError(err)

	if err == nil {
		logEntry.Debug("updateWebhookHealth update")
		status, message = metav1.ConditionTrue, ConditionHealthyDesc
	} else {
		logEntry.Error("updateWebhookHealth update")
		status, message = metav1.ConditionFalse, err.Error()
	}

	webhook.Status = []metav1.Condition{
		{
			Type:               ConditionHealthy,
			Reason:             reason,
			Status:             status,
			Message:            message,
			LastTransitionTime: metav1.NewTime(timestamp),
		},
	}
	go func() {
		s.outUpdatesChan <- webhook
	}()
}

func (s *ControllerState) lazyClient() (client *kubernetes.Clientset, err error) {
	if s.cli != nil {
		client = s.cli
		return
	}

	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return
	}

	if client, err = kubernetes.NewForConfig(config); err == nil {
		s.cli = client
	}

	return
}

func (s *ControllerState) retrieveConfigValue(ctx context.Context, src *api.SecurityEventWebhookConfigVarSource) (string, error) {
	cli, err := s.lazyClient()
	if err != nil {
		return "", err
	}

	if src.ConfigMapKeyRef != nil {
		cm, err := cli.CoreV1().ConfigMaps(ConfigVarNamespace).Get(ctx, src.ConfigMapKeyRef.Name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		if value, present := cm.Data[src.ConfigMapKeyRef.Key]; present {
			return value, nil
		}
		return "", fmt.Errorf("key '%s' not found in the ConfigMap '%s'", src.ConfigMapKeyRef.Key, src.ConfigMapKeyRef.Name)
	} else if src.SecretKeyRef != nil {
		secret, err := cli.CoreV1().Secrets(ConfigVarNamespace).Get(ctx, src.SecretKeyRef.Name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		if value, present := secret.Data[src.SecretKeyRef.Key]; present {
			return string(value), nil
		}
		return "", fmt.Errorf("key '%s' not found in the Secret '%s'", src.SecretKeyRef.Key, src.SecretKeyRef.Name)
	}

	return "", errors.New("neither ConfigMap nor Secret reference present") // should never happen
}

func (s *ControllerState) extractLabels(webhook api.SecurityEventWebhook) map[string]string {
	labels := make(map[string]string)
	if annotation, ok := webhook.Annotations[WebhookLabelsAnnotation]; ok {
		for _, label := range strings.Split(annotation, ",") {
			if keyValue := strings.SplitN(label, ":", 2); len(keyValue) == 2 {
				labels[keyValue[0]] = keyValue[1]
			}
		}
	}
	return labels
}

func (s *ControllerState) debugProcessFunc(webhook *api.SecurityEventWebhook) ProcessFunc {
	return func(context.Context, map[string]string, map[string]string, *lsApi.Event) error {
		logrus.WithField("uid", webhook.UID).Info("Processing Security Events for a webhook in 'Debug' state")
		return nil
	}
}

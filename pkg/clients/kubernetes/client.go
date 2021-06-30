package kubernetes

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	log "github.com/sirupsen/logrus"

	httpCommon "github.com/tigera/es-gateway/pkg/clients/internal/http"
)

// client is a wrapper for the K8s client.
type client struct {
	*kubernetes.Clientset
}

// Client is an interface that exposes the required Kube API operations for ES Gateway.
type Client interface {
	GetSecret(context.Context, string, string) (*v1.Secret, error)
	GetK8sReadyz() error
}

func NewClient(configPath string) (Client, error) {
	// Create a rest.Config. If this runs in k8s, it uses the credentials at fixed locations, otherwise, it
	// uses flags.
	var (
		cfg *rest.Config
		err error
	)

	if len(configPath) == 0 {
		cfg, err = rest.InClusterConfig()
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", configPath)
	}

	if err != nil {
		log.Fatalf("Failed to get k8s cfg %s", err)
	}

	// NewK8sClientWithConfig configures K8s client based a rest.Config.
	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to configure k8s client %s", err)
	}

	return &client{k8s}, nil
}

// GetSecret attempts to retrieve a K8s secret with the given name, from the given namespace
func (c *client) GetSecret(ctx context.Context, name, namespace string) (*v1.Secret, error) {
	secret, err := c.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// GetK8sReadyz checks the readyz endpoint of the Kube API that the client is connected to.
// If the response is anything other than "ok", then an error is returned.
// Otherwise, we return nil.
// http://www.elastic.co/guide/en/elasticsearch/reference/master/cluster-health.html
func (c *client) GetK8sReadyz() error {
	path := "/readyz"
	content, err := c.Discovery().RESTClient().Get().Timeout(httpCommon.HealthCheckTimeout).AbsPath(path).DoRaw(context.TODO())
	if err != nil {
		return err
	}

	contentStr := string(content)
	if contentStr != "ok" {
		return fmt.Errorf(contentStr)
	}

	return nil
}

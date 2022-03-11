package podtemplate

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	elasticGatewayHost           = "tigera-elasticsearch.tigera-secure-es-gateway-http.svc"
	elasticPort                  = 9200
	ElasticsearchADJobUserSecret = "tigera-ee-ad-job-elasticsearch-access"
)

// podTemplateQueryInterface has functionalities to query existing PodTemplates in the cluster
type ADPodTemplateQuery interface {
	GetPodTemplate(ctx context.Context, namespace, name string) (*v1.PodTemplate, error)
}

// podTemplateQueryInstance is the default implementation of podTemplateQueryInterface.
type podTemplateQueryInstance struct {
	k8sClient kubernetes.Interface
}

func NewPodTemplateQuery(k8sClient kubernetes.Interface) ADPodTemplateQuery {
	podTemplateQueryingInstance := &podTemplateQueryInstance{
		k8sClient: k8sClient,
	}

	return podTemplateQueryingInstance
}

func (p *podTemplateQueryInstance) GetPodTemplate(ctx context.Context, namespace, podTemplateName string) (*v1.PodTemplate, error) {
	pt, err := p.k8sClient.CoreV1().PodTemplates(namespace).Get(ctx, podTemplateName, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}
	return pt, nil
}

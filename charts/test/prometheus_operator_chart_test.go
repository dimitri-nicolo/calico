package charttest

import (
	"path/filepath"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestPrometheusOperatorChart(t *testing.T) {
	t.Run("image pull secrets", func(t *testing.T) {
		t.Run("using imagePullSecrets field", func(t *testing.T) {
			opts := &helm.Options{
				SetValues: map[string]string{
					"imagePullSecrets.my-secret": "secret1",
				},
			}

			t.Run("sets imagePullSecrets on deployment", func(t *testing.T) {
				var deployment appsv1.Deployment
				err := renderPrometheusChartResource(t, opts, "templates/04-deployment-prometheus-operator.yaml", &deployment)
				require.NoError(t, err)
				require.ElementsMatch(t, deployment.Spec.Template.Spec.ImagePullSecrets, []corev1.LocalObjectReference{{Name: "my-secret"}})
			})

			t.Run("creates a secret", func(t *testing.T) {
				var secret corev1.Secret
				require.NoError(t, renderPrometheusChartResource(t, opts, "templates/01-imagepullsecret-tigera-prometheus.yaml", &secret))
				require.Equal(t, "my-secret", secret.Name)
				require.Equal(t, map[string][]byte{".dockerconfigjson": []byte("secret1")}, secret.Data)
			})
		})

		t.Run("using imagePullSecretReferences field", func(t *testing.T) {
			opts := &helm.Options{
				SetValues: map[string]string{
					"imagePullSecretReferences[0].name": "my-secret",
				},
			}

			t.Run("sets imagePullSecrets on deployment", func(t *testing.T) {
				var deployment appsv1.Deployment
				err := renderPrometheusChartResource(t, opts, "templates/04-deployment-prometheus-operator.yaml", &deployment)
				require.NoError(t, err)
				require.ElementsMatch(t, deployment.Spec.Template.Spec.ImagePullSecrets, []corev1.LocalObjectReference{{Name: "my-secret"}})
			})

			t.Run("does not create a secret", func(t *testing.T) {
				err := renderPrometheusChartResource(t, opts, "templates/01-imagepullsecret-tigera-prometheus.yaml", nil)
				require.ErrorContains(t, err, "could not find template templates/01-imagepullsecret-tigera-prometheus.yaml in chart")
			})

		})
	})
}

func renderPrometheusChartResource(t *testing.T, options *helm.Options, templatePath string, into any) error {
	t.Helper()
	helmChartPath, err := filepath.Abs("../tigera-prometheus-operator")
	require.NoError(t, err)

	output, err := helm.RenderTemplateE(t, options, helmChartPath, "tigera-prometheus-operator", []string{templatePath})
	if err != nil {
		return err
	}
	helm.UnmarshalK8SYaml(t, output, &into)
	return nil
}

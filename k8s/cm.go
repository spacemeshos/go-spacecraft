package k8s

import (
	"context"

	"helm.sh/helm/v3/pkg/repo"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	helm "github.com/mittwald/go-helm-client"
)

func (k8s *Kubernetes) DeployChaosMesh() error {
	namespaceClient := k8s.Client.CoreV1().Namespaces()

	namespace := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "chaos-testing",
		},
	}

	if _, err := namespaceClient.Create(context.TODO(), namespace, metav1.CreateOptions{}); err != nil {
		return err
	}

	opt := &helm.RestConfClientOptions{
		Options: &helm.Options{
			Debug:     true,
			Linting:   true,
			Namespace: "chaos-testing",
		},
		RestConfig: k8s.RestConfig,
	}

	client, err := helm.NewClientFromRestConf(opt)
	if err != nil {
		return err
	}

	chartRepo := repo.Entry{
		Name: "chaos-mesh",
		URL:  "https://charts.chaos-mesh.org",
	}

	if err := client.AddOrUpdateChartRepo(chartRepo); err != nil {
		return err
	}

	ingressSpec := helm.ChartSpec{
		ReleaseName: "chaos-mesh",
		ChartName:   "chaos-mesh/chaos-mesh",
		Namespace:   "chaos-testing",
		Wait:        true,
		Force:       true,
		Version:     config.ChaosMeshVersion,
		ValuesYaml: sanitizeYaml(`
			chaosDaemon:
				runtime: containerd
				socketPath: /run/containerd/containerd.sock
		`),
	}

	_, err = client.InstallOrUpgradeChart(context.Background(), &ingressSpec, &helm.GenericHelmOptions{})
	return err
}

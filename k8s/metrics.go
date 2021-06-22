package k8s

import (
	"context"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	helm "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"
)

func (k8s *Kubernetes) DeployPrometheus() error {
	opt := &helm.RestConfClientOptions{
		Options: &helm.Options{
			Debug:   true,
			Linting: true,
		},
		RestConfig: k8s.RestConfig,
	}

	client, err := helm.NewClientFromRestConf(opt)
	if err != nil {
		return err
	}

	chartRepo := repo.Entry{
		Name: "ingress-nginx",
		URL:  "https://kubernetes.github.io/ingress-nginx",
	}

	if err := client.AddOrUpdateChartRepo(chartRepo); err != nil {
		return err
	}

	elasticSearchSpec := helm.ChartSpec{
		ReleaseName: "ingress-nginx",
		ChartName:   "ingress-nginx/ingress-nginx",
		Namespace:   "kube-system",
		Wait:        true,
		Force:       true,
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &elasticSearchSpec); err != nil {
		return err
	}

	namespaceClient := k8s.Client.CoreV1().Namespaces()

	namespace := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "monitoring",
		},
	}

	if _, err = namespaceClient.Create(context.TODO(), namespace, metav1.CreateOptions{}); err != nil {
		return err
	}

	chartRepo = repo.Entry{
		Name: "spacemesh",
		URL:  "https://spacemeshos.github.io/ws-helm-charts",
	}

	if err := client.AddOrUpdateChartRepo(chartRepo); err != nil {
		return err
	}

	promSpec := helm.ChartSpec{
		ReleaseName: "prom",
		ChartName:   "spacemesh/sm-prom",
		Namespace:   "monitoring",
		Wait:        true,
		Force:       true,
		ValuesYaml: sanitizeYaml(fmt.Sprintf(`
			kube-prometheus-stack:
			  alertmanager:
				ingress:
				  hosts:
					- alertmanager-132.spacemesh.io
			  grafana:
				ingress:
				  hosts:
					- grafana-132.spacemesh.io
			  prometheus:
				ingress:
				  hosts:
					- prometheus-132.spacemesh.io
			prometheus-pushgateway:
			  ingress:
				hosts:
				  - pushgateway-132.spacemesh.io
		`)),
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &promSpec); err != nil {
		return err
	}

	return nil
}

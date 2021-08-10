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
	namespaceClient := k8s.Client.CoreV1().Namespaces()

	namespace := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "monitoring",
		},
	}

	if _, err := namespaceClient.Create(context.TODO(), namespace, metav1.CreateOptions{}); err != nil {
		return err
	}

	opt := &helm.RestConfClientOptions{
		Options: &helm.Options{
			Debug:     true,
			Linting:   true,
			Namespace: "monitoring",
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
		Version:     "3.32.0",
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &elasticSearchSpec); err != nil {
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
		ValuesYaml: sanitizeYaml(fmt.Sprintf(`
			kube-prometheus-stack:
				alertmanager:
					ingress:
				  		hosts:
								- alertmanager-%s.spacemesh.io
				grafana:
					ingress:
						hosts:
							- grafana-%s.spacemesh.io
				prometheus:
					ingress:
							hosts:
								- prometheus-%s.spacemesh.io
					prometheusSpec:
						resources:
							requests:
								memory: %sGi
								cpu: %s

			prometheus-pushgateway:
				ingress:
					hosts:
						- pushgateway-%s.spacemesh.io
		`, config.NetworkName, config.NetworkName, config.NetworkName, config.PrometheusMemory, config.PrometheusCPU, config.NetworkName)),
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &promSpec); err != nil {
		return err
	}

	return nil
}

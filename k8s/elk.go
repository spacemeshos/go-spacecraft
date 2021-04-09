package k8s

import (
	"context"
	"fmt"

	helm "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"
)

func (k8s *Kubernetes) DeployELK() error {
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
		Name: "elastic",
		URL:  "https://helm.elastic.co",
	}

	// Add a chart-repository to the client
	if err := client.AddOrUpdateChartRepo(chartRepo); err != nil {
		return err
	}

	filebeatSpec := helm.ChartSpec{
		ReleaseName: "filebeat",
		ChartName:   "elastic/filebeat",
		Namespace:   "default",
		UpgradeCRDs: true,
		Wait:        true,
	}

	err = client.InstallOrUpgradeChart(context.Background(), &filebeatSpec)

	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

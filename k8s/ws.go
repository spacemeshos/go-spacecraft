package k8s

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	gabs "github.com/Jeffail/gabs/v2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	helm "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"
)

func (k8s *Kubernetes) DeployWS() error {
	namespaceClient := k8s.Client.CoreV1().Namespaces()

	namespace := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ws",
		},
	}

	if _, err := namespaceClient.Create(context.TODO(), namespace, metav1.CreateOptions{}); err != nil {
		return err
	}

	opt := &helm.RestConfClientOptions{
		Options: &helm.Options{
			Debug:     true,
			Linting:   true,
			Namespace: "ws",
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

	ingressSpec := helm.ChartSpec{
		ReleaseName: "ingress-nginx",
		ChartName:   "ingress-nginx/ingress-nginx",
		Namespace:   "kube-system",
		Wait:        true,
		Force:       true,
		Version:     "3.34.0",
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &ingressSpec); err != nil {
		return err
	}

	chartRepo = repo.Entry{
		Name: "spacemesh",
		URL:  "https://spacemeshos.github.io/ws-helm-charts",
	}

	if err := client.AddOrUpdateChartRepo(chartRepo); err != nil {
		return err
	}

	imageSplit := strings.Split(config.GoSmImage, ":")

	tag := ""

	if len(imageSplit) == 2 {
		tag = imageSplit[1]
	} else {
		tag = "latest"
	}

	respository := imageSplit[0]

	minerConfigBuf, err := ioutil.ReadFile(config.MinerGoSmConfig)

	if err != nil {
		return err
	}

	minerConfigStr := string(minerConfigBuf)
	minerConfigJson, err := gabs.ParseJSON([]byte(minerConfigStr))

	if err != nil {
		return err
	}

	minerPeersBuf, err := ioutil.ReadFile(config.PeersFile)

	if err != nil {
		return err
	}

	minerPeersStr := string(minerPeersBuf)

	networkId := minerConfigJson.Path("p2p.network-id").Data().(float64)

	spacemeshAPISpec := helm.ChartSpec{
		ReleaseName: "spacemesh-api",
		ChartName:   "spacemesh/spacemesh-api",
		Namespace:   "ws",
		ValuesYaml: sanitizeYaml(fmt.Sprintf(`
			image:
				repository: %s
				tag: %s
			netID: %s
			pagerdutyToken: "cfac67a53df9440ad0e5e0fcfe1933db"
			ingress:
				grpcDomain: api-%s.spacemesh.io
				jsonRpcDomain: api-json-%s.spacemesh.io
			config: |
				%s
			peers: |
				%s
		`, respository, tag, fmt.Sprint(networkId), config.NetworkName, config.NetworkName, strings.ReplaceAll(minerConfigStr, "\n", ""), minerPeersStr)),
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &spacemeshAPISpec); err != nil {
		return err
	}

	return nil
}

package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/spacemeshos/go-spacecraft/gcp"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cloudflare "github.com/cloudflare/cloudflare-go"
	helm "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"
)

type Network struct {
	NetName              string  `json:"netName"`
	NetID                float64 `json:"netID"`
	Conf                 string  `json:"conf"`
	GrpcApi              string  `json:"grpcAPI"`
	JsonApi              string  `json:"jsonAPI"`
	Explorer             string  `json:"explorer"`
	ExplorerAPI          string  `json:"explorerAPI"`
	ExplorerVersion      string  `json:"explorerVersion"`
	ExplorerConf         string  `json:"explorerConf"`
	Dash                 string  `json:"dash"`
	DashApi              string  `json:"dashAPI"`
	DashVersion          string  `json:"dashVersion"`
	Repository           string  `json:"repository"`
	MinNodeVersion       string  `json:"minNodeVersion"`
	MaxNodeVersion       string  `json:"maxNodeVersion"`
	MinSmappRelease      string  `json:"minSmappRelease"`
	LatestSmappRelease   string  `json:"latestSmappRelease"`
	SmappBaseDownloadUrl string  `json:"smappBaseDownloadUrl"`
	NodeBaseDownloadUrl  string  `json:"nodeBaseDownloadUrl"`
}

func (k8s *Kubernetes) DeployWS() error {
	certData, err := ioutil.ReadFile(config.TLSCert)

	if err != nil {
		return err
	}

	keyData, err := ioutil.ReadFile(config.TLSKey)

	if err != nil {
		return err
	}

	tlsSecret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tls",
		},
		Data: map[string][]byte{
			"tls.crt": certData,
			"tls.key": keyData,
		},
		Type: "kubernetes.io/tls",
	}

	secretsClient := k8s.Client.CoreV1().Secrets("ws")
	_, err = secretsClient.Create(context.Background(), tlsSecret, metav1.CreateOptions{})

	if err != nil {
		return err
	}

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

	minerConfigStr, err := gcp.ReadConfig()

	if err != nil {
		return err
	}

	spacemeshAPISpec := helm.ChartSpec{
		ReleaseName: "spacemesh-api",
		ChartName:   "spacemesh/spacemesh-api",
		Namespace:   "ws",
		ValuesYaml: sanitizeYaml(fmt.Sprintf(`
			resources:
				requests:
					memory: %sGi
					cpu: %s
			image:
				repository: %s
				tag: %s
			pagerdutyToken: "cfac67a53df9440ad0e5e0fcfe1933db"
			ingress:
				grpcDomain: api-%s.spacemesh.io
				jsonRpcDomain: api-json-%s.spacemesh.io
			config: |
				%s
		`, config.MinerMemory, config.MinerCPU, respository, tag, config.NetworkName, config.NetworkName, strings.ReplaceAll(minerConfigStr, "\n", ""))),
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &spacemeshAPISpec); err != nil {
		return err
	}

	spacemeshExplorerSpec := helm.ChartSpec{
		ReleaseName: "spacemesh-explorer",
		ChartName:   "spacemesh/spacemesh-explorer",
		Namespace:   "ws",
		ValuesYaml: sanitizeYaml(fmt.Sprintf(`
			resources:
				requests:
					memory: %sGi
					cpu: %s
			imageTag: %s
			apiServer:
				ingress:
					domain: explorer-api-%s.spacemesh.io
			node:
				image:
					repository: %s
					tag: %s
				config: |
					%s
		`, config.MinerMemory, config.MinerCPU, config.ExplorerVersion, config.NetworkName, respository, tag, strings.ReplaceAll(minerConfigStr, "\n", ""))),
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &spacemeshExplorerSpec); err != nil {
		return err
	}

	spacemeshDashSpec := helm.ChartSpec{
		ReleaseName: "spacemesh-dash",
		ChartName:   "spacemesh/spacemesh-dash",
		Namespace:   "ws",
		ValuesYaml: sanitizeYaml(fmt.Sprintf(`
			mongo: mongodb://spacemesh-explorer-mongo
			ingress:
				domain: dash-api-%s.spacemesh.io
			image:
				tag: %s
		`, config.NetworkName, config.DashboardVersion)),
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &spacemeshDashSpec); err != nil {
		return err
	}

	if config.CloudflareAPIToken != "" {
		ingressClient := k8s.Client.ExtensionsV1beta1().Ingresses("ws")

		for range time.Tick(5 * time.Second) {
			ingress, err := ingressClient.Get(context.Background(), "spacemesh-api", metav1.GetOptions{})

			if err != nil {
				return err
			}

			fmt.Println("waiting for ingress")

			if len(ingress.Status.LoadBalancer.Ingress) == 1 {
				ip := ingress.Status.LoadBalancer.Ingress[0].IP

				api, err := cloudflare.NewWithAPIToken(config.CloudflareAPIToken)

				if err != nil {
					return err
				}

				id, err := api.ZoneIDByName("spacemesh.io")

				if err != nil {
					return err
				}

				_, err = api.CreateDNSRecord(context.Background(), id, cloudflare.DNSRecord{
					Type:    "A",
					Name:    "api-json-" + config.NetworkName + ".spacemesh.io",
					Content: ip,
				})

				if err != nil {
					return err
				}

				_, err = api.CreateDNSRecord(context.Background(), id, cloudflare.DNSRecord{
					Type:    "A",
					Name:    "api-" + config.NetworkName + ".spacemesh.io",
					Content: ip,
				})

				if err != nil {
					return err
				}

				proxied := true

				_, err = api.CreateDNSRecord(context.Background(), id, cloudflare.DNSRecord{
					Type:    "A",
					Name:    "dash-api-" + config.NetworkName + ".spacemesh.io",
					Content: ip,
					Proxied: &proxied,
				})

				if err != nil {
					return err
				}

				_, err = api.CreateDNSRecord(context.Background(), id, cloudflare.DNSRecord{
					Type:    "A",
					Name:    "explorer-api-" + config.NetworkName + ".spacemesh.io",
					Content: ip,
					Proxied: &proxied,
				})

				if err != nil {
					return err
				}

				break
			}
		}
	}

	return nil
}

func (k8s *Kubernetes) AddToDiscovery() error {
	networksConfig, err := gcp.ReadWSConfig()

	if err != nil {
		return err
	}

	var networks []Network

	json.Unmarshal([]byte(networksConfig), &networks)

	imageSplit := strings.Split(config.GoSmImage, ":")

	tag := ""
	image := ""

	if len(imageSplit) == 2 {
		image = imageSplit[0]
		tag = imageSplit[1]
	} else {
		tag = "latest"
	}

	minerConfigStr, err := gcp.ReadConfig()

	if err != nil {
		return err
	}

	minerConfigJson, err := gabs.ParseJSON([]byte(minerConfigStr))

	if err != nil {
		return err
	}

	netID, ok := minerConfigJson.Path("p2p.network-id").Data().(float64)

	if !ok {
		return errors.New("cannot read p2p.network-id from config file")
	}

	network := Network{
		NetName:              config.NetworkName,
		NetID:                netID,
		Conf:                 "https://storage.googleapis.com/spacecraft-data/" + config.NetworkName + "-archive/config.json",
		GrpcApi:              "https://api-" + config.NetworkName + ".spacemesh.io/",
		JsonApi:              "https://api-json-" + config.NetworkName + ".spacemesh.io/",
		Explorer:             "https://explorer.spacemesh.io/",
		ExplorerAPI:          "https://explorer-api-" + config.NetworkName + ".spacemesh.io/",
		ExplorerVersion:      config.ExplorerVersion,
		ExplorerConf:         "https://storage.googleapis.com/spacecraft-data/" + config.NetworkName + "-archive/config.json",
		Dash:                 "https://dash.spacemesh.io/",
		DashApi:              "wss://dash-api-" + config.NetworkName + ".spacemesh.io/ws",
		DashVersion:          config.DashboardVersion,
		Repository:           image,
		MinNodeVersion:       tag,
		MaxNodeVersion:       tag,
		MinSmappRelease:      config.SmappVersion,
		LatestSmappRelease:   config.SmappVersion,
		SmappBaseDownloadUrl: "https://downloads.spacemesh.io",
		NodeBaseDownloadUrl:  "https://downloads.spacemesh.io",
	}

	networks = append(networks, network)

	json, err := json.MarshalIndent(networks, "", "  ")

	if err != nil {
		return err
	}

	err = gcp.UploadWSConfig(string(json))

	if err != nil {
		return err
	}

	return nil
}

func (k8s *Kubernetes) RemoveFromDiscovery() error {
	networksConfig, err := gcp.ReadWSConfig()

	if err != nil {
		return err
	}

	var networks []Network
	var newNetworks []Network

	json.Unmarshal([]byte(networksConfig), &networks)

	for _, network := range networks {
		if network.NetName != config.NetworkName {
			newNetworks = append(newNetworks, network)
		}
	}

	json, err := json.MarshalIndent(newNetworks, "", "  ")

	if err != nil {
		return err
	}

	if string(json) == "null" {
		err = gcp.UploadWSConfig(string("[]"))

		if err != nil {
			return err
		}
	} else {
		err = gcp.UploadWSConfig(string(json))

		if err != nil {
			return err
		}
	}

	return nil
}

func (k8s *Kubernetes) DeleteWSDNSRecords() error {
	if config.CloudflareAPIToken != "" {
		api, err := cloudflare.NewWithAPIToken(config.CloudflareAPIToken)

		if err != nil {
			return err
		}

		id, err := api.ZoneIDByName("spacemesh.io")

		if err != nil {
			return err
		}

		records, err := api.DNSRecords(context.Background(), id, cloudflare.DNSRecord{
			Name: "api-json-" + config.NetworkName + ".spacemesh.io",
		})

		if err != nil {
			return err
		}

		if len(records) == 1 {
			err = api.DeleteDNSRecord(context.Background(), id, records[0].ID)

			if err != nil {
				return err
			}
		}

		records, err = api.DNSRecords(context.Background(), id, cloudflare.DNSRecord{
			Name: "api-" + config.NetworkName + ".spacemesh.io",
		})

		if err != nil {
			return err
		}

		if len(records) == 1 {
			err = api.DeleteDNSRecord(context.Background(), id, records[0].ID)

			if err != nil {
				return err
			}
		}

		records, err = api.DNSRecords(context.Background(), id, cloudflare.DNSRecord{
			Name: "dash-api-" + config.NetworkName + ".spacemesh.io",
		})

		if err != nil {
			return err
		}

		if len(records) == 1 {
			err = api.DeleteDNSRecord(context.Background(), id, records[0].ID)

			if err != nil {
				return err
			}
		}

		records, err = api.DNSRecords(context.Background(), id, cloudflare.DNSRecord{
			Name: "explorer-api-" + config.NetworkName + ".spacemesh.io",
		})

		if err != nil {
			return err
		}

		if len(records) == 1 {
			err = api.DeleteDNSRecord(context.Background(), id, records[0].ID)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

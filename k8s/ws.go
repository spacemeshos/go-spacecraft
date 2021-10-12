package k8s

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cloudflare "github.com/cloudflare/cloudflare-go"
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

	spacemeshAPISpec := helm.ChartSpec{
		ReleaseName: "spacemesh-api",
		ChartName:   "spacemesh/spacemesh-api",
		Namespace:   "ws",
		ValuesYaml: sanitizeYaml(fmt.Sprintf(`
			image:
				repository: %s
				tag: %s
			pagerdutyToken: "cfac67a53df9440ad0e5e0fcfe1933db"
			ingress:
				grpcDomain: api-%s.spacemesh.io
				jsonRpcDomain: api-json-%s.spacemesh.io
			config: |
				%s
		`, respository, tag, config.NetworkName, config.NetworkName, strings.ReplaceAll(minerConfigStr, "\n", ""))),
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &spacemeshAPISpec); err != nil {
		return err
	}

	spacemeshExplorerSpec := helm.ChartSpec{
		ReleaseName: "spacemesh-explorer",
		ChartName:   "spacemesh/spacemesh-explorer",
		Namespace:   "ws",
		ValuesYaml: sanitizeYaml(fmt.Sprintf(`
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
		`, config.ExplorerVersion, config.NetworkName, respository, tag, strings.ReplaceAll(minerConfigStr, "\n", ""))),
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

				proxied := true

				_, err = api.CreateDNSRecord(context.Background(), id, cloudflare.DNSRecord{
					Type:    "A",
					Name:    "api-json-" + config.NetworkName + ".spacemesh.io",
					Content: ip,
					Proxied: &proxied,
				})

				if err != nil {
					return err
				}

				_, err = api.CreateDNSRecord(context.Background(), id, cloudflare.DNSRecord{
					Type:    "A",
					Name:    "api-" + config.NetworkName + ".spacemesh.io",
					Content: ip,
					Proxied: &proxied,
				})

				if err != nil {
					return err
				}

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

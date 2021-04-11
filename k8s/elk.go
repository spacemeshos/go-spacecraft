package k8s

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	helm "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"
)

func sanitizeYaml(yaml string) string {
	return strings.ReplaceAll(yaml, "	", "  ")
}

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

	if err := client.AddOrUpdateChartRepo(chartRepo); err != nil {
		return err
	}

	elasticSearchSpec := helm.ChartSpec{
		ReleaseName: "elasticsearch",
		ChartName:   "elastic/elasticsearch",
		Namespace:   "default",
		Wait:        true,
		Force:       true,
		ValuesYaml: sanitizeYaml(`
			volumeClaimTemplate:
				accessModes: [ "ReadWriteOnce" ]
				resources:
					requests:
						storage: 30Gi
		`),
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &elasticSearchSpec); err != nil {
		return err
	}

	logstashSpec := helm.ChartSpec{
		ReleaseName: "logstash",
		ChartName:   "elastic/logstash",
		Namespace:   "default",
		Wait:        true,
		Force:       true,
		ValuesYaml: sanitizeYaml(`
			persistence:
				enabled: true
	
			logstashConfig:
				logstash.yml: |
					http.host: 0.0.0.0
					xpack.monitoring.enabled: false
	
			logstashPipeline:
				uptime.conf: |
					input { beats { port => 5044 } }
					filter {
						json{
							source => "message"
							target => "sm"
							skip_on_invalid_json => false
						}
	
						mutate {
							add_field => { "name" => "%{[kubernetes][labels][name]}" }
	
							remove_field => [
								"log",
								"cloud",
								"ecs",
								"agent",
								"input",
								"tags",
								"docker",
								"container",
								"host",
								"message",
								"[sm][T]",
								"kubernetes"
							]
						}
					}
					output { elasticsearch { hosts => ["http://elasticsearch-master:9200"] index => "sm-logs" manage_template => false } }
	
			service:
				annotations: {}
				type: ClusterIP
				loadBalancerIP: ""
				ports:
					- name: beats
						port: 5044
						protocol: TCP
						targetPort: 5044
					- name: http
						port: 8080
						protocol: TCP
						targetPort: 8080
		`),
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &logstashSpec); err != nil {
		return err
	}

	kibanaSpec := helm.ChartSpec{
		ReleaseName: "kibana",
		ChartName:   "elastic/kibana",
		Namespace:   "default",
		Wait:        true,
		Force:       true,
		SkipCRDs:    true,
		UpgradeCRDs: false,
		ValuesYaml: sanitizeYaml(`
			service:
				type: LoadBalancer
		`),
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &kibanaSpec); err != nil {
		if !strings.Contains(err.Error(), "failed to replace object") {
			return err
		}
	}

	filebeatSpec := helm.ChartSpec{
		ReleaseName: "filebeat",
		ChartName:   "elastic/filebeat",
		Namespace:   "default",
		Wait:        true,
		Force:       true,
		ValuesYaml: sanitizeYaml(`
			daemonset:
				filebeatConfig:
					filebeat.yml: |
						filebeat.autodiscover:
							providers:
								- type: kubernetes
									templates:
										- condition.contains:
												kubernetes.container.name: miner
											config:
												- type: docker
													containers.ids:
														- "${data.kubernetes.container.id}"
										- condition.contains:
												kubernetes.container.name: poet
											config:
												- type: docker
													containers.ids:
														- "${data.kubernetes.container.id}"
						output.logstash:
							hosts: 'logstash-logstash:5044'
		`),
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &filebeatSpec); err != nil {
		return err
	}

	kibanaURL, err := k8s.GetKibanaURL()

	if err != nil {
		return err
	}

	url := "http://" + kibanaURL + "/api/saved_objects/_import"
	method := "POST"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	file, err := os.Open(config.KibanaSavedObjects)
	if err != nil {
		return err
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(config.KibanaSavedObjects))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	httpClient := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		return err
	}
	req.Header.Add("kbn-xsrf", "true")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err = httpClient.Do(req)

	if err != nil {
		return err
	}

	return nil
}

func (k8s *Kubernetes) GetKibanaURL() (string, error) {
	services, err := k8s.Client.CoreV1().Services("default").List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return "", err
	}

	for _, svc := range services.Items {
		if svc.Name == "kibana-kibana" {
			return svc.Status.LoadBalancer.Ingress[0].IP + ":5601", nil
		}
	}

	return "", errors.New("Kibana URL not found")
}

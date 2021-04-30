package k8s

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	apiv1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	helm "github.com/mittwald/go-helm-client"
	"github.com/sethvargo/go-password/password"
	"helm.sh/helm/v3/pkg/repo"
)

func sanitizeYaml(yaml string) string {
	return strings.ReplaceAll(yaml, "	", "  ")
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (k8s *Kubernetes) DeployELK() error {
	pass, err := password.Generate(32, 10, 0, false, false)
	if err != nil {
		return err
	}

	k8s.Password = pass

	secretsClient := k8s.Client.CoreV1().Secrets("default")
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "elastic-credentials",
		},
		StringData: map[string]string{
			"username": "elastic",
			"password": pass,
		},
	}
	_, err = secretsClient.Create(context.Background(), secret, metav1.CreateOptions{})

	if err != nil {
		return err
	}

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
		ValuesYaml: sanitizeYaml(fmt.Sprintf(`
			replicas: 1
			minimumMasterNodes: 1
			service:
				type: NodePort
			volumeClaimTemplate:
				accessModes: [ "ReadWriteOnce" ]
				resources:
					requests:
						storage: %sGi
			resources:
				requests:
					cpu: "%s"
					memory: "%sGi"
				limits:
					cpu: "%s"
					memory: "%sGi"
			esConfig:
				elasticsearch.yml: |
					xpack.security.enabled: true
			extraEnvs:
				- name: ELASTIC_PASSWORD
					valueFrom:
						secretKeyRef:
							name: elastic-credentials
							key: password
				- name: ELASTIC_USERNAME
					valueFrom:
						secretKeyRef:
							name: elastic-credentials
							key: username
		`, config.ESDiskSize, config.ESCPU, config.ESMemory, config.ESCPU, config.ESMemory)),
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
		ValuesYaml: sanitizeYaml(fmt.Sprintf(`
			logstashConfig:
				logstash.yml: |
					http.host: 0.0.0.0
					xpack.monitoring.enabled: false
	
			logstashPipeline:
				uptime.conf: |
					input { beats { port => 5044 } }
					filter {
						if [message] =~ "\A\{.+\}\z" {
							json {
								source => "message"
								target => "sm"
								skip_on_invalid_json => false
							}
			
							mutate {
								remove_field => [
									"message"
								]
							}
						}
			
						mutate {
							add_field => { "name" => "%%{[kubernetes][labels][name]}" }
			
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
								"[sm][T]",
								"kubernetes"
							]
						}
					}
					output { elasticsearch { hosts => ["http://elasticsearch-master:9200"] index => "sm-%%{+YYYY.MM.dd}" manage_template => false user => '${ELASTICSEARCH_USERNAME}' password => '${ELASTICSEARCH_PASSWORD}' } }
				
			extraEnvs:
				- name: 'ELASTICSEARCH_USERNAME'
					valueFrom:
						secretKeyRef:
							name: elastic-credentials
							key: username
				- name: 'ELASTICSEARCH_PASSWORD'
					valueFrom:
						secretKeyRef:
							name: elastic-credentials
							key: password
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
			
			resources:
				requests:
					cpu: "%s"
					memory: "%sGi"
				limits:
					cpu: "%s"
					memory: "%sGi"

			volumeClaimTemplate:
				accessModes: [ "ReadWriteOnce" ]
				resources:
					requests:
						storage: %sGi
		`, config.LogstashCPU, config.LogstashMemory, config.LogstashCPU, config.LogstashMemory, config.LogstashDiskSize)),
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
		ValuesYaml: sanitizeYaml(fmt.Sprintf(`
			service:
				type: LoadBalancer

			resources:
				requests:
					cpu: "%s"
					memory: "%sGi"
				limits:
					cpu: "%s"
					memory: "%sGi"
			extraEnvs:
				- name: 'ELASTICSEARCH_USERNAME'
					valueFrom:
						secretKeyRef:
							name: elastic-credentials
							key: username
				- name: 'ELASTICSEARCH_PASSWORD'
					valueFrom:
						secretKeyRef:
							name: elastic-credentials
							key: password
		`, config.KibanaCPU, config.KibanaMemory, config.KibanaCPU, config.KibanaMemory)),
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
				resources: {}
				filebeatConfig:
					filebeat.yml: |
						processors:
							- script:
									lang: javascript
									id: my_filter
									source: >
										function process(event) {
											var message = event.Get('message')
											try {
												var msg = JSON.parse(message)
												Object.keys(msg).forEach(function(k) {msg[k] = msg[k].toString()})
												event.Put("message", JSON.stringify(msg));
											} catch(e) {}
										}
						filebeat:
							autodiscover.providers:
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
	req.Header.Add("Authorization", "Basic "+basicAuth("elastic", pass))

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

func (k8s *Kubernetes) GetESURL() (string, error) {
	port, err := k8s.GetExternalPort("elasticsearch-master", "http")

	if err != nil {
		return "", err
	}

	ip, err := k8s.GetExternalIP()

	if err != nil {
		return "", err
	}

	return ip + ":" + port, nil

}

func (k8s *Kubernetes) SetupLogDeletionPolicy() error {
	esURL, err := k8s.GetESURL()

	httpClient := &http.Client{}

	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, "http://"+esURL+"/_ilm/policy/cleanup-history", bytes.NewBuffer([]byte(fmt.Sprintf("{\"policy\":{\"phases\":{\"hot\":{\"actions\":{}},\"delete\":{\"min_age\":\"%sd\",\"actions\":{\"delete\":{}}}}}}", config.LogsExpiry))))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+basicAuth("elastic", k8s.Password))

	resp, err := httpClient.Do(req)

	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return err
		}

		return errors.New(string(body))
	}

	req, err = http.NewRequest(http.MethodPut, "http://"+esURL+"/sm-*/_settings?pretty", bytes.NewBuffer([]byte("{\"lifecycle.name\":\"cleanup-history\"}")))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+basicAuth("elastic", k8s.Password))

	resp, err = httpClient.Do(req)

	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return err
		}

		return errors.New(string(body))
	}

	req, err = http.NewRequest(http.MethodPut, "http://"+esURL+"/_template/logging_policy_template?pretty", bytes.NewBuffer([]byte("{\"index_patterns\":[\"sm-*\"],\"settings\":{\"index.lifecycle.name\":\"cleanup-history\"}}")))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+basicAuth("elastic", k8s.Password))

	resp, err = httpClient.Do(req)

	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return err
		}

		return errors.New(string(body))
	}

	return nil
}

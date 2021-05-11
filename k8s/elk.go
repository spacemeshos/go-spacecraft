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

	certData, err := ioutil.ReadFile(config.ESCert)

	if err != nil {
		return err
	}

	secret = &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "elastic-certificates",
		},
		Data: map[string][]byte{
			"elastic-certificates.p12": certData,
		},
	}
	_, err = secretsClient.Create(context.Background(), secret, metav1.CreateOptions{})

	if err != nil {
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
			clusterHealthCheckParams: 'wait_for_status=yellow&timeout=1s'
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
					xpack.security.transport.ssl.enabled: true
					xpack.security.transport.ssl.verification_mode: certificate
					xpack.security.transport.ssl.keystore.path: /usr/share/elasticsearch/config/certs/elastic-certificates.p12
					xpack.security.transport.ssl.truststore.path: /usr/share/elasticsearch/config/certs/elastic-certificates.p12
			imageTag: "7.12.1"
			secretMounts:
			- name: elastic-certificates
				secretName: elastic-certificates
				path: /usr/share/elasticsearch/config/certs
			extraEnvs:
				- name: ES_HEAP_SIZE
					value: %sg
				- name: ES_JAVA_OPTS
					value: -Xmx%sg -Xms%sg
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
		`, config.ESDiskSize, config.ESCPU, config.ESMemory, config.ESCPU, config.ESMemory, config.ESHeapMemory, config.ESHeapMemory, config.ESHeapMemory)),
	}

	if err = client.InstallOrUpgradeChart(context.Background(), &elasticSearchSpec); err != nil {
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
												event.Put("name", event.Get('kubernetes.labels.name'))
												delete msg.T
												var sm = {}
												Object.keys(msg).forEach(function(k) { sm[k] = msg[k] })
												event.Put("sm", sm)
												event.Delete("message")
											} catch(e) {
												var message = event.Get('message')
												var sm = { message: message }
												event.Delete("message")
												event.Put("sm", sm);
												event.Put("name", event.Get('kubernetes.labels.name'))
											}
										}
							- drop_fields:
									fields: ["log", "cloud", "ecs", "agent", "input", "tags", "docker", "container", "host", "kubernetes"]
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
						output.elasticsearch:
							host: '${NODE_NAME}'
							hosts: '${ELASTICSEARCH_HOSTS:elasticsearch-master:9200}'
							username: "${ELASTICSEARCH_USERNAME}"
							password: "${ELASTICSEARCH_PASSWORD}"
							index: "sm-%{+YYYY.MM.dd}"
							worker: 3
							bulk_max_size: 1000
						setup.template.enabled: false
						setup.ilm.enabled: false
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

package k8s

import (
	"context"
	"strings"

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

	return nil
}

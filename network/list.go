package network

import (
	"errors"
	"fmt"

	gabs "github.com/Jeffail/gabs/v2"
	"github.com/spacemeshos/go-spacecraft/gcp"
	k8s "github.com/spacemeshos/go-spacecraft/k8s"
	"github.com/spacemeshos/go-spacecraft/log"
)

func ListNetworks() error {
	networks, err := gcp.GetClusters()

	if err != nil {
		return err
	}

	if len(networks) == 0 {
		log.Error.Println("No networks found")
	} else {
		log.Info.Println("Here is the list of deployed networks:")
		for _, name := range networks {
			k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient(name)

			if err != nil {
				return err
			}

			kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

			pyroscopeURL, err := kubernetes.GetPyroscopeURL()

			if err != nil {
				return err
			}

			kibanaPassword, err := kubernetes.GetKibanaPassword()

			if err != nil {
				return err
			}

			configFile, err := gcp.ReadConfig(name)
			if err != nil {
				return err
			}

			configJson, err := gabs.ParseJSON([]byte(configFile))
			if err != nil {
				return err
			}

			netID, ok := configJson.Path("p2p.network-id").Data().(float64)

			if !ok {
				return errors.New("cannot read network-id")
			}

			image, err := kubernetes.GetMinerImage("miner-1")
			if err != nil {
				return err
			}

			log.Success.Print("\nNetwork Name: " + name)
			fmt.Printf(`
NETID: %s
Kibana URL: https://kibana-%s.spacemesh.io
Kibana Password: %s
Grafana URL: https://grafana-%s.spacemesh.io
Grafana Username: admin
Grafana Password: prom-operator
Prometheus URL: https://prometheus-%s.spacemesh.io
Pyroscope URL: http://%s
Config: https://storage.googleapis.com/spacecraft-data/%s-archive/config.json
Docker URL: %s`,
				fmt.Sprintf("%v", netID),
				name,
				kibanaPassword,
				name,
				name,
				pyroscopeURL,
				name,
				image,
			)
		}
	}

	return nil
}

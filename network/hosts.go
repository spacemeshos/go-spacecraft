package network

import (
	"fmt"
	"strings"

	"github.com/spacemeshos/go-spacecraft/gcp"
	k8s "github.com/spacemeshos/go-spacecraft/k8s"
	"github.com/spacemeshos/go-spacecraft/log"
)

func ListHosts() error {
	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient(config.NetworkName)

	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

	miners, err := kubernetes.GetMiners()

	if err != nil {
		return err
	}

	apiURLs := []string{}

	ip, err := kubernetes.GetExternalIP()

	if err != nil {
		return err
	}

	log.Info.Println("API URLs: ")

	for _, miner := range miners {
		port, err := kubernetes.GetExternalPort(miner, "grpcport")
		if err != nil {
			return err
		}

		fmt.Println(miner + ":" + ip + ":" + port)

		apiURLs = append(apiURLs, ip+":"+port+"/"+miner)
	}

	log.Info.Println("Spacemesh Watch: ")
	fmt.Println(strings.Join(apiURLs[:], ","))

	return nil
}

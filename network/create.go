package network

import (
	"fmt"

	"github.com/spacemeshos/spacecraft/gcp"
)

func Create() error {
	err := gcp.CreateK8SCluster()

	if err != nil {
		return err
	}

	k8s, err := gcp.GetKubernetesClient()

	if err != nil {
		return err
	}

	fmt.Println(k8s)

	return nil
}

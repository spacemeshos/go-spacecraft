package network

import (
	"fmt"

	"github.com/spacemeshos/go-spacecraft/gcp"
)

func ListNetworks() error {
	networks, err := gcp.GetClusters()

	if err != nil {
		return err
	}

	if len(networks) == 0 {
		fmt.Println("No networks found")
	} else {
		fmt.Println("Here is the list of deployed networks:")
		for _, name := range networks {
			fmt.Println(name)
		}
	}

	return nil
}

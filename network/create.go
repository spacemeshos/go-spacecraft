package network

import gcp "github.com/spacemeshos/spacecraft/gcp"

func Create() {
	gcp.CreateK8SCluster()
}

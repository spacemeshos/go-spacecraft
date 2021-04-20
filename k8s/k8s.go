package k8s

import (
	"sync"

	cfg "github.com/spacemeshos/go-spacecraft/config"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

var config = &cfg.Config

type Kubernetes struct {
	Client      *kubernetes.Clientset
	RestConfig  *restclient.Config
	CurrentNode int
	mu          sync.Mutex
	Password    string
}

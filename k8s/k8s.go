package k8s

import (
	"k8s.io/client-go/kubernetes"
)

type Kubernetes struct {
	client *kubernetes.Clientset
}

func (k8s *Kubernetes) execueCommand() string {
	return ""
}

func (k8s *Kubernetes) createDeployment() (string, error) {
	return "", nil
}

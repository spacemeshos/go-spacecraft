package network

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	pb "github.com/spacemeshos/api/release/go/spacemesh/v1"
	"github.com/spacemeshos/go-spacecraft/gcp"
	k8s "github.com/spacemeshos/go-spacecraft/k8s"
	"github.com/spacemeshos/go-spacecraft/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func Rewards() error {
	if config.Host == "" {
		return errors.New("You need to specify the host")
	}

	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient(config.NetworkName)

	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

	managedAddresses, err := kubernetes.MinerAccounts()

	if err != nil {
		return err
	}

	conn, err := grpc.Dial(config.Host, grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		return err
	}

	c := pb.NewDebugServiceClient(conn)
	r, err := c.Accounts(context.Background(), &emptypb.Empty{})

	if err != nil {
		return err
	}

	for _, account := range r.AccountWrapper {
		log.Info.Println("Account: " + hex.EncodeToString(account.AccountId.Address))
		fmt.Println("Balance: " + strconv.FormatUint(account.StateCurrent.Balance.Value, 10))
		_, exists := find(managedAddresses, hex.EncodeToString(account.AccountId.Address))
		fmt.Println("Managed Miner Account: ", strconv.FormatBool(exists)+"\n")
	}

	return nil
}

func find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

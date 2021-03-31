package cmd

import (
	"fmt"

	"github.com/spacemeshos/spacecraft/log"
	"github.com/spacemeshos/spacecraft/network"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var createNetworkCmd = &cobra.Command{
	Use:   "createNetwork",
	Short: "Create a network",
	Long: `Create a new network with N miners and N PoETs. For example:

spacecraft createNetwork -m=10 -p=3`,
	Run: func(cmd *cobra.Command, args []string) {
		err := network.Create()
		if err != nil {
			log.Error.Println(err)
		}
		// ctx, _ := context.WithCancel(context.Background())
		// <-ctx.Done()
	},
}

func init() {
	rootCmd.AddCommand(createNetworkCmd)

	createNetworkCmd.Flags().StringVar(&config.GoSmConfig, "go-sm-config", config.GoSmConfig, "config file for go-spacemesh")
	createNetworkCmd.Flags().StringVar(&config.PoetConfig, "poet-config", config.PoetConfig, "config file for poet")
	createNetworkCmd.Flags().IntVarP(&config.NumberOfMiners, "miners", "m", config.NumberOfMiners, "number of miners")
	createNetworkCmd.Flags().IntVarP(&config.NumberOfPoets, "poets", "p", config.NumberOfPoets, "number of poets")
	createNetworkCmd.Flags().StringVar(&config.MinerMemory, "miner-ram", config.MinerMemory, "RAM for each miner")
	createNetworkCmd.Flags().StringVar(&config.MinerCPU, "miner-cpu", config.MinerCPU, "vCPUs for each miner")
	createNetworkCmd.Flags().StringVar(&config.PoetMemory, "poet-ram", config.PoetMemory, "RAM for each poet")
	createNetworkCmd.Flags().StringVar(&config.PoetCPU, "poet-cpu", config.PoetCPU, "vCPUs for each poet")
	createNetworkCmd.Flags().StringVar(&config.MinerDiskSize, "miner-disk-size", config.MinerDiskSize, "Disk size of miner in GB")
	createNetworkCmd.Flags().StringVar(&config.PoetDiskSize, "poet-disk-size", config.PoetDiskSize, "Disk size of poet in GB")
	createNetworkCmd.Flags().StringVar(&config.GoSmImage, "go-sm-image", config.GoSmImage, "docker image for go-spacemesh build")
	createNetworkCmd.Flags().StringVar(&config.PoetImage, "poet-image", config.PoetImage, "docker image for poet build")

	err := viper.BindPFlags(createNetworkCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}

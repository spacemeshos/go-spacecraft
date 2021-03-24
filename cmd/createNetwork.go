package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var createNetworkCmd = &cobra.Command{
	Use:   "createNetwork",
	Short: "Create a network",
	Long: `Create a new network with N miners and N PoETs. For example:

spacecraft createNetwork -m=10 -p=3`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.NetworkName)
	},
}

func init() {
	rootCmd.AddCommand(createNetworkCmd)

	createNetworkCmd.Flags().IntVarP(&config.NumberOfMiners, "miners", "m", config.NumberOfMiners, "number of miners")
	createNetworkCmd.Flags().IntVarP(&config.NumberOfPoets, "poets", "p", config.NumberOfPoets, "number of poets")
	createNetworkCmd.Flags().IntVar(&config.MinerMemory, "miner-ram", config.MinerMemory, "RAM for each miner")
	createNetworkCmd.Flags().IntVar(&config.MinerCPU, "miner-cpu", config.MinerCPU, "vCPUs for each miner")
	createNetworkCmd.Flags().IntVar(&config.PoetMemory, "poet-ram", config.PoetMemory, "RAM for each poet")
	createNetworkCmd.Flags().IntVar(&config.PoetCPU, "poet-cpu", config.PoetCPU, "vCPUs for each poet")
	createNetworkCmd.Flags().IntVar(&config.MinerDiskSize, "miner-disk-size", config.MinerDiskSize, "Disk size of miner in GB")
	createNetworkCmd.Flags().IntVar(&config.PoetDiskSize, "poet-disk-size", config.PoetDiskSize, "Disk size of poet in GB")
	createNetworkCmd.Flags().StringVar(&config.GoSmImage, "go-sm-image", config.GoSmImage, "docker image for go-spacemesh build")
	createNetworkCmd.Flags().StringVar(&config.PoetImage, "poet-image", config.PoetImage, "docker image for poet build")

	err := viper.BindPFlags(createNetworkCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}

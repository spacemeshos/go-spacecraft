package cmd

import (
	"fmt"

	"github.com/spacemeshos/spacecraft/config"
	"github.com/spf13/cobra"
)

var createNetworkCmd = &cobra.Command{
	Use:   "createNetwork",
	Short: "Create a network",
	Long: `Create a new network with N miners and N PoETs. For example:

spacecraft createNetwork -m=10 -p=3`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("createNetwork called")
	},
}

func init() {
	rootCmd.AddCommand(createNetworkCmd)

	createNetworkCmd.Flags().IntVarP(&config.NumberOfMiners, "miners", "m", 10, "number of miners")
	createNetworkCmd.Flags().IntVarP(&config.NumberOfPoets, "poets", "p", 3, "number of poets")
	createNetworkCmd.Flags().IntVar(&config.MinerMemory, "miner-ram", 4, "RAM for each miner")
	createNetworkCmd.Flags().IntVar(&config.MinerCPU, "miner-cpu", 2, "vCPUs for each miner")
	createNetworkCmd.Flags().IntVar(&config.PoetMemory, "poet-ram", 4, "RAM for each poet")
	createNetworkCmd.Flags().IntVar(&config.PoetCPU, "poet-cpu", 2, "vCPUs for each poet")
}

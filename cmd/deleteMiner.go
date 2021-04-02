package cmd

import (
	"fmt"

	"github.com/spacemeshos/spacecraft/log"
	"github.com/spacemeshos/spacecraft/network"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deleteMinerCmd = &cobra.Command{
	Use:   "deleteMiner",
	Short: "Delete a miner",
	Long: `Delete a individual miner from the network. For example:

spacecraft deleteMiner --miner=1`,
	Run: func(cmd *cobra.Command, args []string) {
		err := network.DeleteMiner()
		if err != nil {
			log.Error.Println(err)
		}

		log.Success.Println("miner delete successfully")
	},
}

func init() {
	rootCmd.AddCommand(deleteMinerCmd)

	deleteMinerCmd.Flags().StringVar(&config.MinerNumber, "miner-number", config.MinerNumber, "miner to delete")

	err := viper.BindPFlags(deleteMinerCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}

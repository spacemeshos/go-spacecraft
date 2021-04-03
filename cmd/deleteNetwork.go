package cmd

import (
	"github.com/spacemeshos/spacecraft/log"
	"github.com/spacemeshos/spacecraft/network"
	"github.com/spf13/cobra"
)

var deleteNetworkCmd = &cobra.Command{
	Use:   "deleteNetwork",
	Short: "Delete a network",
	Run: func(cmd *cobra.Command, args []string) {
		err := network.Delete()
		if err != nil {
			log.Error.Println(err)
			return
		}

		log.Success.Println("network deleted successfully")
	},
}

func init() {
	rootCmd.AddCommand(deleteNetworkCmd)
}

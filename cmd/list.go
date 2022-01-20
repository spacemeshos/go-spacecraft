package cmd

import (
	"github.com/spacemeshos/go-spacecraft/log"
	"github.com/spacemeshos/go-spacecraft/network"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Get the list of networks",
	Run: func(cmd *cobra.Command, args []string) {
		err := network.ListNetworks()
		if err != nil {
			log.Error.Println(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

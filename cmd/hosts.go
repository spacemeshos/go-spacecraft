package cmd

import (
	"github.com/spacemeshos/go-spacecraft/log"
	"github.com/spacemeshos/go-spacecraft/network"
	"github.com/spf13/cobra"
)

var hostsCmd = &cobra.Command{
	Use:   "hosts",
	Short: "List GRPC endpoints of miners",
	Run: func(cmd *cobra.Command, args []string) {
		err := network.ListHosts()
		if err != nil {
			log.Error.Println(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(hostsCmd)
}

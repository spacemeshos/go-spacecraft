package cmd

import (
	"github.com/spacemeshos/go-spacecraft/log"
	"github.com/spacemeshos/go-spacecraft/network"
	"github.com/spf13/cobra"
)

var grpcCmd = &cobra.Command{
	Use:   "viewAPI",
	Short: "List GRPC endpoints of miners",
	Run: func(cmd *cobra.Command, args []string) {
		err := network.APIURLs()
		if err != nil {
			log.Error.Println(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(grpcCmd)
}

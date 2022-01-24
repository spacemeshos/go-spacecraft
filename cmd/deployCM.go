package cmd

import (
	"fmt"

	"github.com/spacemeshos/go-spacecraft/log"
	"github.com/spacemeshos/go-spacecraft/network"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deployCMCmd = &cobra.Command{
	Use:   "deployCM",
	Short: "Deploys chaos mesh",
	Long:  `For example: spacecraft deployCM`,
	Run: func(cmd *cobra.Command, args []string) {
		err := network.DeployCM()
		if err != nil {
			log.Error.Println(err)
			return
		}

		log.Success.Println("chaos mesh deployed successfully")
	},
}

func init() {
	rootCmd.AddCommand(deployCMCmd)
	deployCMCmd.Flags().StringVar(&config.ChaosMeshVersion, "chaos-mesh-version", config.ChaosMeshVersion, "chaosmesh version")

	err := viper.BindPFlags(deployCMCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}

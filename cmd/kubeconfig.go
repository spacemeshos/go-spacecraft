package cmd

import (
	"fmt"

	"github.com/spacemeshos/go-spacecraft/gcp"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	rootCmd.AddCommand(&kubeconfigCmd)
}

var kubeconfigCmd = cobra.Command{
	Use:   "kubeconfig",
	Short: "Generate kubeconfig",
	Long: `Examples:
// print to stdout
spacecraft kubeconfig
// save to a file
spacecraft kubeconfig devnet101`,
	RunE: func(cmd *cobra.Command, args []string) error {
		conf, err := gcp.GetClientConfig()
		if err != nil {
			return err
		}
		raw, err := conf.RawConfig()
		if err != nil {
			return err
		}
		if len(args) == 1 {
			return clientcmd.WriteToFile(raw, args[0])
		}
		buf, err := clientcmd.Write(raw)
		if err != nil {
			return err
		}
		_, err = fmt.Printf("%s", buf)
		return err
	},
}

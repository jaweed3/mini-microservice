package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "order-cli",
	Short: "Order service CLI",
	Long:  "order-cli is a service to ordering seblak from restaurants straight to your house",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("order-cli")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "oops. seems your seblak may encountered some error, '%s'\n", err)
		os.Exit(1)
	}
}

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [id]",
	Short: "check order status",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := http.Get(BaseURL + "/order/detail?id=" + args[0])
		if err != nil {
			fmt.Printf("failed: %s\n", err)
			return
		}
		defer resp.Body.Close()
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		fmt.Printf("%v\n", out)
	},
}

func init() { rootCmd.AddCommand(statusCmd) }

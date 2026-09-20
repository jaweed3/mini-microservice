package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

func CheckMenu() {
	resp, err := http.Get(BaseURL + "/menu")
	if err != nil {
		log.Printf("failed to send request : %s\n", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("server returned status : %d\n", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read response: %s\n", err)
		return
	}

	var menu map[string]int
	if err := json.Unmarshal(body, &menu); err != nil {
		log.Printf("failed to parse response: %s\n", err)
		return
	}
	for k, v := range menu {
		fmt.Printf("%s: %d\n", k, v)
	}
}

var checkMenu = &cobra.Command{
	Use:     "menu",
	Aliases: []string{"menu"},
	Short:   "check menu of restaurant",
	Long:    "check whats avaliable in our restaurant menu",
	Run: func(cmd *cobra.Command, args []string) {
		CheckMenu()
	},
}

func init() {
	rootCmd.AddCommand(checkMenu)
}

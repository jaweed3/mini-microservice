package cmd

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"
)

func OrderSeblak(name string, amount int) {
	req := OrderRequest{Name: name, Amount: amount}
	body, _ := json.Marshal(req)
	resp, err := http.Post(BaseURL+"/order", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("failed to send request : %s\n", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("server returned %d", resp.StatusCode)
		return
	}

	var out OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		log.Printf("failed to parse response : %s\n", err)
		return
	}
	log.Printf("ID: %v\n Amount: %v\n Message: %s", out.ID, out.Amount, out.Message)
}

var orderCmd = &cobra.Command{
	Use:     "order [name] [amount]",
	Aliases: []string{"order"},
	Short:   "Order seblak package",
	Long:    "Order seblak package with amount you needed.",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		amount, err := strconv.Atoi(args[1])
		if err != nil || amount <= 0 {
			log.Printf("invalid amount %q", args[1])
			return
		}
		OrderSeblak(args[0], amount)
	},
}

func init() {
	rootCmd.AddCommand(orderCmd)
}

type OrderRequest struct {
	Name   string `json:"name"`
	Amount int    `json:"amount"`
}

type OrderResponse struct {
	ID        int    `json:"id"`
	EventID   string `json:"event_id"`
	RequestID string `json:"request_id"`
	Name      string `json:"name"`
	Amount    int    `json:"amount"`
	Message   string `json:"message"`
}

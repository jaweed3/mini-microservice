// Package contract : definition for contract struct types across projects.
package contract

type OrderEvent struct {
	EventID   string `json:"event_id"`
	RequestID string `json:"request_id"`
	OrderID   int    `json:"order_id"`
	Name      string `json:"name"`
	Amount    int    `json:"amount"`
	Action    string `json:"action"`
	Status    string `json:"status"`
}

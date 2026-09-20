package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jaweed3/mini-microservice/contract"
)

type Order struct {
	ID     int32
	Name   string
	Amount int
	Status string
}

type SQLOrderRepository struct {
	db *sql.DB
}

// OrderRepository decouple the service from concrete db
type OrderRepository interface {
	// new order service
	CreateNewOrder(ctx context.Context, order *Order, reqID string) error
	// get by id service
	GetByID(ctx context.Context, id int) (*Order, error)
	// update status order
	UpdateStatus(ctx context.Context, id int, status string, reqID string) error
}

// OrderService define the repo .
type OrderService struct {
	repo OrderRepository
}

// NewOrderService is constructor for inject repo
func NewOrderService(repo OrderRepository) *OrderService {
	return &OrderService{
		repo: repo,
	}
}

func NewSQLOrderRepository(db *sql.DB) *SQLOrderRepository {
	return &SQLOrderRepository{db: db}
}

func (s *OrderService) UpdateStatus(ctx context.Context, id int, status string, reqID string) error {
	if !isValidStatus(status) {
		return errors.New("invalid status")
	}
	return s.repo.UpdateStatus(ctx, id, status, reqID)
}

func isValidStatus(status string) bool {
	switch status {
	case "PENDING", "PAID", "CANCELLED", "DONE":
		return true
	}
	return false
}

func (s *OrderService) GetOrderID(ctx context.Context, id int) (*Order, error) {
	return s.repo.GetByID(ctx, id)
}

// InsertNewOrder create new order and store it to database
func (s *OrderService) InsertNewOrder(ctx context.Context, name string, amount int, reqID string) (*Order, error) {
	if amount <= 0 || amount > 30 {
		return nil, errors.New("your order amount doesnt make sense")
	}

	order := &Order{Name: name, Amount: amount}
	if err := s.repo.CreateNewOrder(ctx, order, reqID); err != nil {
		return nil, err
	}

	return order, nil
}

func (r *SQLOrderRepository) CreateNewOrder(ctx context.Context, order *Order, reqID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	query := `
	INSERT INTO orders (name, amount, status)
	VALUES ($1, $2, 'PENDING')
	RETURNING id, status
	`

	if err := tx.QueryRowContext(ctx, query, order.Name, order.Amount).
		Scan(&order.ID, &order.Status); err != nil {
		return fmt.Errorf("error when insert order : %s", err)
	}

	event := contract.OrderEvent{
		EventID:   uuid.New().String(),
		RequestID: reqID,
		OrderID:   int(order.ID),
		Name:      order.Name,
		Amount:    order.Amount,
		Action:    "CREATE",
		Status:    "PENDING",
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error when serialize the order: %s", err)
	}

	_, err = tx.ExecContext(ctx, `
	  INSERT INTO outbox (event_id, subject, payload)
		VALUES ($1, $2, $3)
	`, event.EventID, "order.created", string(payload))
	if err != nil {
		return fmt.Errorf("error when insert outbox: %s", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error when trying to commit transaction: %s", err)
	}

	log.Printf("order_id [%d]: order successfully placed: ", order.ID)
	return nil
}

func (r *SQLOrderRepository) GetByID(ctx context.Context, id int) (*Order, error) {
	var o Order
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id,name,amount,status FROM orders WHERE id=$1`,
		id,
	).Scan(&o.ID, &o.Name, &o.Amount, &o.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("order %d not found: %w", id, err)
		}
		return nil, err
	}

	return &o, nil
}

func (r *SQLOrderRepository) UpdateStatus(
	ctx context.Context,
	id int,
	status string,
	reqID string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to beginTx : %w", err)
	}
	defer tx.Rollback()

	var o Order
	err = tx.QueryRowContext(ctx,
		`UPDATE orders SET status=$1 WHERE id=$2
	  RETURNING id, name, amount, status
		`,
		status, id).Scan(&o.ID, &o.Name, &o.Amount, &o.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("order %d cannot found", id)
		}
		return fmt.Errorf("failed to update status: %w", err)
	}

	event := contract.OrderEvent{
		EventID:   uuid.NewString(),
		RequestID: reqID,
		Name:      o.Name,
		Amount:    o.Amount,
		OrderID:   id,
		Action:    "UPDATE_STATUS",
		Status:    o.Status,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error when serialize the order: %s", err)
	}

	_, err = tx.ExecContext(ctx, `
	INSERT INTO outbox (event_id, subject, payload)
	VALUES ($1, $2, $3)
	`, event.EventID, "order.updated", string(payload))
	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}

	return tx.Commit()
}

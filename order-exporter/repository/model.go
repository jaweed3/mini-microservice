// Package repository : for
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

type ExportedOrder struct {
	ID        int
	EventID   string
	OrderID   int
	Name      string
	Amount    int
	Action    string
	Status    string
	CreatedAt time.Time
}

// ExportedRepository definition (repository)
type ExportedRepository interface {
	Insert(ctx context.Context, order *ExportedOrder) error
	UpdateStatus(ctx context.Context, orderID int, eventID string, status string) error
}

// ExportedService definition (service)
type ExportedService struct {
	exportedRepo ExportedRepository
}

// NewOrderExport definition (new repo for returning service)
func NewOrderExport(exportRepo ExportedRepository) *ExportedService {
	return &ExportedService{
		exportedRepo: exportRepo,
	}
}

// InsertNewOrderExport definition (service, need definition for business case.)
func (exportService *ExportedService) InsertNewOrderExport(
	ctx context.Context,
	orderID int,
	name string,
	amount int,
	action string,
	status string,
	eventID string,
) (*ExportedOrder, error) {
	exportOrder := &ExportedOrder{
		EventID: eventID,
		OrderID: orderID,
		Name:    name,
		Amount:  amount,
		Status:  status,
		Action:  action,
	}

	if err := exportService.exportedRepo.Insert(ctx, exportOrder); err != nil {
		log.Printf("error occured when trying to insert : \n%v", err)
		return nil, err
	}

	return exportOrder, nil
}

func (exportService *ExportedService) UpdateOrderStatus(
	ctx context.Context,
	orderID int,
	eventID string,
	status string,
) error {
	if !isValidStatus(status) {
		return fmt.Errorf("invalid status: %s", status)
	}
	return exportService.exportedRepo.UpdateStatus(ctx, orderID, eventID, status)
}

func isValidStatus(status string) bool {
	switch status {
	case "PENDING", "PAID", "CANCELLED", "DONE":
		return true
	}
	return false
}

type SQLExportRepository struct {
	db *sql.DB
}

func NewSQLExportRepository(db *sql.DB) *SQLExportRepository {
	return &SQLExportRepository{db: db}
}

func (repo *SQLExportRepository) UpdateStatus(ctx context.Context, orderID int, eventID string, status string) error {
	res, err := repo.db.ExecContext(ctx, `
		UPDATE exported_orders SET status = $1 WHERE order_id = $2`,
		status, orderID,
	)
	if err != nil {
		return fmt.Errorf("update status : %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected : %w", err)
	}

	if n == 0 {
		_, err = repo.db.ExecContext(ctx, `
		INSERT INTO exported_orders(event_id,order_id,name,amount,action,status)
		VALUES ($1, $2, '',0 ,'UPDATE_STATUS', $3) ON CONFLICT DO NOTHING
		`, eventID, orderID, status)
		if err != nil {
			return fmt.Errorf("error occured when inserting event to exported_orders: %s", err)
		}
		return nil
	}
	return nil
}

func (repo *SQLExportRepository) Insert(ctx context.Context, exportedOrder *ExportedOrder) error {
	query := `INSERT INTO exported_orders (
		event_id, 
		order_id, 
		name, 
		amount,
		status,
		action
	) VALUES (
		$1, $2, $3, $4, $5, $6
	) RETURNING id, created_at`

	err := repo.db.QueryRowContext(
		ctx,
		query,
		exportedOrder.EventID,
		exportedOrder.OrderID,
		exportedOrder.Name,
		exportedOrder.Amount,
		exportedOrder.Status,
		exportedOrder.Action,
	).Scan(&exportedOrder.ID, &exportedOrder.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			log.Printf("duplicate event %s , skipping\n", exportedOrder.EventID)
			return nil
		}
		return fmt.Errorf("failed to create insert to db : \n%v", err)
	}

	log.Print("query successfully executed. :)")
	return nil
}

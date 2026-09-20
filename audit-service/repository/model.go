package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type AuditLog struct {
	ID        int32
	EventID   string
	RequestID string
	OrderID   int
	Name      string
	Amount    int
	Action    string
	Status    string
	CreatedAt time.Time
}

type AuditRepository interface {
	// new service for log audit from order service
	CreateNewLog(ctx context.Context, auditLog *AuditLog) error
}

type AuditService struct {
	auditRepo AuditRepository
}

func NewLogAudit(auditRepo AuditRepository) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
	}
}

func (auditService *AuditService) InsertNewAuditLog(
	ctx context.Context,
	orderID int,
	requestID string,
	name string,
	amount int,
	action string,
	status string,
	eventID string,
) (*AuditLog, error) {
	auditLog := &AuditLog{
		EventID:   eventID,
		RequestID: requestID,
		OrderID:   orderID,
		Name:      name,
		Amount:    amount,
		Status:    status,
		Action:    action,
	}
	if err := auditService.auditRepo.CreateNewLog(ctx, auditLog); err != nil {
		return nil, err
	}

	return auditLog, nil
}

type SQLAuditRepository struct {
	db *sql.DB
}

func NewSQLAuditRepository(db *sql.DB) *SQLAuditRepository {
	return &SQLAuditRepository{db: db}
}

func (repo *SQLAuditRepository) CreateNewLog(ctx context.Context, auditLog *AuditLog) error {
	query := `INSERT INTO audit_service (event_id, request_id, order_id, name, amount, action, status) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (event_id) DO NOTHING
		RETURNING id, created_at`

	err := repo.db.QueryRowContext(
		ctx,
		query,
		auditLog.EventID,
		auditLog.RequestID,
		auditLog.OrderID,
		auditLog.Name,
		auditLog.Amount,
		auditLog.Action,
		auditLog.Status,
	).Scan(&auditLog.ID, &auditLog.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("failed to create log audit : %w", err)
	}
	return nil
}

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

type AuditLog struct {
	ID        int32
	EventID   string
	OrderID   int
	Name      string
	Amount    int
	Action    string
	CreatedAt time.Time
}

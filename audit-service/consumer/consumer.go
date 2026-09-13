// Package consumer is for make event and consume nats subject :)
package consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jaweed3/mini-microservice/audit-service/repository"
	"github.com/jaweed3/mini-microservice/contract"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type AuditConsumer struct {
	nc      *nats.Conn
	service *repository.AuditService
}

func NewAuditConsumer(nc *nats.Conn, svc *repository.AuditService) *AuditConsumer {
	return &AuditConsumer{nc: nc, service: svc}
}

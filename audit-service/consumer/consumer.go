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

func (c *AuditConsumer) Start(ctx context.Context) error {
	js, err := jetstream.New(c.nc)
	if err != nil {
		return err
	}

	stream, err := js.Stream(ctx, "ORDERS")
	if err != nil {
		return err
	}

	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          "audit-consumer",
		Durable:       "audit-consumer",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: "order.>",
		MaxDeliver:    5,
	})
	if err != nil {
		return err
	}

	// kita consume
	_, err = cons.Consume(func(msg jetstream.Msg) {
		processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()

		var evt contract.OrderEvent
		if err := json.Unmarshal(msg.Data(), &evt); err != nil {
			log.Printf("[request_id=%s] invalid message : %v", evt.RequestID, err)
			msg.Term()
			return
		}

		auditLog, err := c.service.InsertNewAuditLog(
			processCtx,
			evt.OrderID,
			evt.RequestID,
			evt.Name,
			evt.Amount,
			evt.Action,
			evt.Status,
			evt.EventID,
		)
		if err != nil {
			log.Printf("[request_id=%s] insert audit log failed : %v", evt.RequestID, err)
			msg.Nak()
			return
		}

		log.Printf("[request_id=%s] insert audit log success : %+v", evt.RequestID, auditLog)
		msg.Ack()
	})
	return err
}

// Package consumer : define consumer for nats request
package consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jaweed3/mini-microservice/contract"
	"github.com/jaweed3/mini-microservice/order-exporter/repository"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type ExportConsumer struct {
	nc      *nats.Conn
	service *repository.ExportedService
}

func NewExportConsumer(nc *nats.Conn, service *repository.ExportedService) *ExportConsumer {
	return &ExportConsumer{
		nc:      nc,
		service: service,
	}
}

func (c *ExportConsumer) Start(ctx context.Context) error {
	js, err := jetstream.New(c.nc)
	if err != nil {
		return err
	}

	stream, err := js.Stream(ctx, "ORDERS")
	if err != nil {
		return err
	}

	if _, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "DLQ_ORDERS",
		Subjects: []string{"dlq.>"},
		Storage:  jetstream.FileStorage,
	}); err != nil {
		return err
	}

	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          "order-export-consumer",
		Durable:       "order-export-consumer",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: "order.>",
		MaxDeliver:    5,
	})
	if err != nil {
		return err
	}

	_, err = cons.Consume(func(msg jetstream.Msg) {
		processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()

		var evt contract.OrderEvent
		if err := json.Unmarshal(msg.Data(), &evt); err != nil {
			c.toDLQ(js, msg, "invalid json: "+err.Error())
			return
		}

		switch msg.Subject() {
		case "order.created":
			// handle create
			orderExport, err := c.service.InsertNewOrderExport(
				processCtx,
				evt.OrderID,
				evt.Name,
				evt.Amount,
				evt.Action,
				evt.Status,
				evt.EventID,
			)
			if err != nil {
				c.handleFailure(js, msg, err, evt.RequestID)
				return
			}

			if orderExport.ID == 0 {
				log.Printf("[request_id=%s] dup %s skip", evt.RequestID, evt.EventID)
				msg.Ack()
				return
			}

			log.Printf("[request_id=%s] insert success : %+v", evt.RequestID, orderExport)
			msg.Ack()

		case "order.updated":
			// handle update
			if err := c.service.UpdateOrderStatus(processCtx, evt.OrderID, evt.EventID, evt.Status); err != nil {
				c.handleFailure(js, msg, err, evt.RequestID)
				return
			}

			log.Printf("[request_id=%s] update success : order=%d status=%s", evt.RequestID, evt.OrderID, evt.Status)
			msg.Ack()

		default:
			log.Printf("unhandled subject: %s", msg.Subject())
			msg.Ack()
		}
	})
	return err
}

func (c *ExportConsumer) handleFailure(
	js jetstream.JetStream,
	msg jetstream.Msg,
	err error,
	reqID string,
) {
	meta, _ := msg.Metadata()

	if meta != nil && meta.NumDelivered >= 5 {
		c.toDLQ(js, msg, err.Error())
		return
	}
	log.Printf("[request_id=%s] retry (delivery %d): %v", reqID, meta.NumDelivered, err)
	msg.Nak()
}

func (c *ExportConsumer) toDLQ(
	js jetstream.JetStream,
	msg jetstream.Msg,
	reason string,
) {
	meta, _ := msg.Metadata()
	n := uint64(0)
	if meta != nil {
		n = meta.NumDelivered
	}

	log.Printf("[DLQ] subject=%s deliveries=%d reason=%s", msg.Subject(), n, reason)

	if _, err := js.Publish(context.Background(), "dlq.orders", msg.Data()); err != nil {
		log.Printf("[DLQ] publish failed: %v (will retry)", err)
		msg.Nak()
		return
	}
	msg.Ack()
}

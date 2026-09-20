package repository

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type OutboxRelay struct {
	db      *sql.DB
	natsURL string
	js      nats.JetStreamContext
}

func NewOutboxRelay(db *sql.DB, natsURL string) *OutboxRelay {
	return &OutboxRelay{db: db, natsURL: natsURL}
}

func (r *OutboxRelay) ensureNATS() bool {
	if r.js != nil {
		return true
	}

	nc, err := nats.Connect(r.natsURL, nats.Timeout(2*time.Second))
	if err != nil {
		log.Printf("relay: NATS not ready : %v", err)
		return false
	}

	js, err := nc.JetStream()
	if err != nil {
		log.Printf("relay: JetStream init : %v", err)
		nc.Close()
		return false
	}

	cfg := &nats.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"order.created", "order.updated"},
		Storage:  nats.FileStorage,
	}
	if _, err := js.AddStream(cfg); err != nil {
		if errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
			if _, err := js.UpdateStream(cfg); err != nil {
				log.Printf("UpdateStream: %v", err)
			}
		} else {
			log.Printf("AddStream: %v", err)
		}
	}

	r.js = js
	log.Println("relay: NATS connected")
	return true
}

func (r *OutboxRelay) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
		r.relayOnce(ctx)
	}
}

func (r *OutboxRelay) relayOnce(ctx context.Context) {
	if !r.ensureNATS() {
		return
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, event_id, subject, payload 
	  FROM outbox WHERE published=0 ORDER BY id LIMIT 50`)
	if err != nil {
		log.Printf("relay query: %s", err)
		return
	}

	type item struct {
		id      int64
		eventID string
		subject string
		payload string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.eventID, &it.subject, &it.payload); err != nil {
			log.Printf("relay scan: %v", err)
			continue
		}

		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		log.Printf("relay rows : %v", err)
	}
	rows.Close()

	for _, it := range items {
		if _, err := r.js.Publish(it.subject,
			[]byte(it.payload), nats.MsgId(it.eventID)); err != nil {
			log.Printf("relay error when publish: %v", err)
			continue
		}
		if _, err := r.db.ExecContext(ctx,
			`UPDATE outbox SET published=1 WHERE id=?`, it.id); err != nil {
			log.Printf("db error execute outbox query, relay mark: %v", err)
		}
	}
}

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jaweed3/mini-microservice/audit-service/consumer"
	"github.com/jaweed3/mini-microservice/audit-service/repository"
	"github.com/nats-io/nats.go"
	_ "modernc.org/sqlite"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./audit.db"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Printf("error when trying to connect to db : %v", err)
		return
	}

	defer db.Close()

	if err := repository.InitDBAudit(db); err != nil {
		log.Fatalf("error when init db for audit  service : \n%v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		fmt.Printf("error when trying to connect to NATS Jetstream!\n%v", err)
		return
	} else {
		fmt.Printf("Subscribing order created to Default URL NATS: \n%v", natsURL)
	}
	defer nc.Close()

	auditRepo := repository.NewSQLAuditRepository(db)
	auditService := repository.NewLogAudit(auditRepo)
	consumer := consumer.NewAuditConsumer(nc, auditService)

	if err := consumer.Start(ctx); err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
	log.Println("shutting down...")

	if err := nc.Drain(); err != nil {
		log.Printf("nats drain err...: \n%v", err)
	}

	if err := db.Close(); err != nil {
		log.Printf("error when trying to close the db : \n%v", err)
	}
	log.Println("bye...")
}

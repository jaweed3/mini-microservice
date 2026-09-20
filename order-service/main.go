package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jaweed3/mini-microservice/order-service/middleware"
	"github.com/jaweed3/mini-microservice/order-service/repository"
	"github.com/nats-io/nats.go"

	_ "modernc.org/sqlite"
)

func getRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Got / Request \n")
	io.WriteString(w, "this is my website!\n")
}

func getOrderID(w http.ResponseWriter, r *http.Request, service *repository.OrderService) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}

	o, err := service.GetOrderID(r.Context(), id)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":     o.ID,
		"name":   o.Name,
		"amount": o.Amount,
		"status": o.Status,
	})
}

func updateStatus(
	w http.ResponseWriter,
	r *http.Request,
	service *repository.OrderService,
) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	reqID := middleware.GetRequestID(r.Context())

	if err := service.UpdateStatus(r.Context(), id, req.Status, reqID); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":      id,
		"status":  req.Status,
		"message": "status updated!",
	})
}

var menu = map[string]int{
	"seblak": 15000,
	"mie":    12000,
	"bakso":  10000,
}

func getMenu(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(menu)
}

func getOrder(
	w http.ResponseWriter,
	r *http.Request,
	service *repository.OrderService,
) {
	reqID := middleware.GetRequestID(r.Context())

	var req struct {
		Name   string `json:"name"`
		Amount int    `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, ok := menu[req.Name]; !ok {
		jsonError(w, "unknown menu: "+req.Name, http.StatusBadRequest)
		return
	}

	// use the service here.
	order, err := service.InsertNewOrder(r.Context(), req.Name, req.Amount, reqID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// make response to the request
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":         order.ID,
		"request_id": reqID,
		"name":       order.Name,
		"amount":     order.Amount,
		"message":    "your order is recorded :)",
	})
}

func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./orders.db"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Error when open db: %s", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)
	for _, p := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(p); err != nil {
			log.Fatalf("pragma %q: %v", p, err)
		}
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		fmt.Printf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()

	// init database schema
	if err := repository.InitDB(db); err != nil {
		log.Fatal(err)
	}

	orderRepo := repository.NewSQLOrderRepository(db)
	orderService := repository.NewOrderService(orderRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", getRoot)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			http.Error(w, "db down", 500)
			return
		}
		w.Write([]byte("ok"))
	})

	getOrderIDHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		getOrderID(w, r, orderService)
	})
	mux.Handle("GET /order/detail", middleware.TraceMiddleware(getOrderIDHandler))

	getMenuHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		getMenu(w, r)
	})
	mux.Handle("GET /menu", middleware.TraceMiddleware(getMenuHandler))

	orderHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		getOrder(w, r, orderService)
	})
	mux.Handle("POST /order", middleware.TraceMiddleware(orderHandler))

	statusHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		updateStatus(w, r, orderService)
	})
	mux.Handle("PATCH /order/status", middleware.TraceMiddleware(statusHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3333"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	relay := repository.NewOutboxRelay(db, natsURL)
	go relay.Run(ctx)

	go func() {
		fmt.Println("server started at :3333")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error : \n%v\n", err)
	}

	log.Println("good bye!!...")
}

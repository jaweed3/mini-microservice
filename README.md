# mini-microservice

Order seblak over HTTP, event-driven via NATS JetStream. SQLite per service.

- `order-service` — HTTP API + outbox relay (`orders.db`)
- `audit-service` — durable consumer, append-only log (`audit.db`)
- `order-exporter` — read-model consumer (`order-exporter.db`)
- `order-cli` — cobra CLI (talks to `localhost:3333`)
- `contract` — shared `OrderEvent`

Flow: `POST /order` writes order + outbox in one txn → relay publishes
`order.created` / `order.updated` on stream `ORDERS` → audit + exporter consume.

## Run local

Needs Go 1.26 and `nats-server`.

```sh
nats-server -js &
cd order-service && PORT=3333 NATS_URL=nats://127.0.0.1:4222 go run .
cd ../audit-service && NATS_URL=nats://127.0.0.1:4222 go run .
cd ../order-exporter && NATS_URL=nats://127.0.0.1:4222 go run .
```

## API

| Method | Path                   | Body                                  |
| ------ | ---------------------- | ------------------------------------- |
| POST   | `/order`               | `{"name":"seblak","amount":2}` → 201  |
| PATCH  | `/order/status?id=1`   | `{"status":"PAID"}`                   |
| GET    | `/order/detail?id=1`   | —                                     |
| GET    | `/menu`                | —                                     |
| GET    | `/healthz`             | —                                     |

Menu: `seblak`, `mie`, `bakso`. Amount 1–30. Status: `PENDING`, `PAID`, `CANCELLED`, `DONE`.
Pass `X-Request-ID` or one is generated and echoed back.

## CLI

```sh
cd order-cli && go run . menu
go run . order seblak 2
go run . status 1
```

## Deploy (k3s via colima)

```sh
make up   # build -> load -> apply -> restart -> status
make logs # follow order-service logs
make pf   # forward localhost:3333
```

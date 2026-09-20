.PHONY: build load apply up restart logs
NS := seblak
K3S := order-service/deploy/k3s

build:
	docker build --build-arg SERVICE=order-service -t order:local .
	docker build --build-arg SERVICE=audit-service -t audit:local .
	docker build --build-arg SERVICE=order-exporter -t exporter:local .

load:
	colima ssh -- sh -c "docker save order:local audit:local exporter:local | sudo ctr -n k8s.io images import -"

apply:
	kubectl -n $(NS) apply -f $(K3S)/

up: build load apply restart status

status:
	kubectl -n $(NS) rollout status deploy/order deploy/audit deploy/exporter --timeout=60s

restart:
	kubectl -n $(NS) rollout restart deploy/order deploy/audit deploy/exporter

logs:
	kubectl -n $(NS) logs -f deploy/order 

pf:
	kubectl -n $(NS) port-forward deploy/order 3333:3333

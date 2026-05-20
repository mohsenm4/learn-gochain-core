.PHONY: up down logs restart clean balance utxos chain mempool build-cli info network

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

restart:
	docker compose restart

clean:
	docker compose down -v

# Quick query helpers. Usage: make balance addr=genesis
balance:
	curl -s localhost:9090/balance/$(addr) | jq .

utxos:
	curl -s localhost:9090/utxos/$(addr) | jq .

chain:
	curl -s localhost:9090/chain | jq '.length, .chain[-1]'

mempool:
	curl -s localhost:9090/mempool | jq .

info:
	curl -s localhost:9090/info | jq .

network:
	curl -s localhost:9090/network | jq .

build-cli:
	go build -o bin/gochain-cli ./cmd/cli

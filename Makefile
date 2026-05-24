.PHONY: up down logs restart clean balance utxos chain mempool build build-cli build-node info network walletinfo mine reset

# Docker
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

# Build local binaries
build: build-node build-cli

build-node:
	go build -o bin/gochain-node ./cmd/node

build-cli:
	go build -o bin/gochain-cli ./cmd/cli

# Reset local state (deletes chain DB and all wallet files in cwd).
reset:
	rm -rf chainDB miner.wallet.json *.wallet.json

# Quick query helpers. Usage: make balance addr=0x...
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

walletinfo:
	curl -s localhost:9090/walletinfo | jq .

mine:
	curl -s -X POST localhost:9090/mine | jq .

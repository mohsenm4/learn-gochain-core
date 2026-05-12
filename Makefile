.PHONY: up down logs restart clean balance utxos chain mempool

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

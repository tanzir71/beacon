.PHONY: check build test lint run clean landing

REPO_SLUG := tanzir71/beacon

check:
	go run ./scripts/gate.go check

build:
	go run ./scripts/gate.go build

test:
	go run ./scripts/gate.go test

lint:
	go run ./scripts/gate.go lint

run:
	go run ./scripts/gate.go run

clean:
	go clean ./...

landing:
	go run ./scripts/landing

.PHONY: test bench lint fuzz build

test:
	go test ./... -count=1 -race

bench:
	go test ./... -bench=. -benchmem -count=5 | tee bench.txt

benchstat: bench
	benchstat bench.txt

lint:
	go vet ./...
	staticcheck ./... 2>/dev/null || true

fuzz:
	go test ./... -fuzz=. -fuzztime=30s

build:
	go build ./...
	go build ./cmd/iso8583
	go build ./cmd/iso8583gen

cover:
	go test ./... -coverprofile=cover.out
	go tool cover -html=cover.out -o cover.html

clean:
	rm -f cover.out cover.html bench.txt iso8583 iso8583gen

.PHONY: all build test clean run studio widget tidy docker

BINARY_NAME=notiflux
VERSION=1.0.0

all: test build

tidy:
	go mod tidy

test:
	go test -v -race -cover ./...

widget:
	cd widget && npm install && npm run build

studio:
	cd web && npm install && npm run build

build:
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/notiflux

run:
	go run ./cmd/notiflux serve --port 8085 --db ./notiflux.db

clean:
	rm -rf bin/ dist/ web/dist/ widget/dist/ notiflux.db*

docker:
	docker build -t notiflux:$(VERSION) -t notiflux:latest .

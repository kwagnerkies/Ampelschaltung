BINARY := ampel
CONFIG := configs/config.yaml
PI_HOST ?= ampel@raspberrypi.local
PI_BIN ?= /usr/local/bin/ampel
PI_STAGE ?= /tmp/ampel-install

.PHONY: all build pi pi64 install test vet fmt lint validate deploy install-pi clean

all: fmt vet test build

build:
	go build -o bin/ ./src/...

# pi baut fuer Raspberry Pi OS 32 Bit, das auf Pi 2 und Pi 3 laeuft. pi64 baut fuer die
# 64-Bit-Variante, die es erst ab Pi 3 gibt.
pi:
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o bin/$(BINARY)-armv7 ./src/ampel
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o bin/ampelctl-armv7 ./src/ampelctl

pi64:
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/$(BINARY)-arm64 ./src/ampel

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

lint:
	golangci-lint run

validate: build
	./bin/$(BINARY) -config $(CONFIG) -validate

deploy: pi
	scp bin/$(BINARY)-armv7 $(PI_HOST):/tmp/$(BINARY)
	ssh $(PI_HOST) 'sudo install -m 0755 /tmp/$(BINARY) $(PI_BIN) && sudo systemctl restart ampel'

install: build
	sudo sh deploy/install.sh .

install-pi: pi
	ssh $(PI_HOST) 'mkdir -p $(PI_STAGE)'
	scp bin/$(BINARY)-armv7 bin/ampelctl-armv7 $(CONFIG) deploy/ampel.service deploy/install.sh \
		docs/aufbau.md docs/vorfuehrung.md $(PI_HOST):$(PI_STAGE)/
	ssh $(PI_HOST) 'sudo sh $(PI_STAGE)/install.sh $(PI_STAGE)'

clean:
	rm -rf bin

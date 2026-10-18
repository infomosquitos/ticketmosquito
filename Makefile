.PHONY: build test vet serve clean build-extensions release-artifacts

VERSION ?= dev

build:
	go build -ldflags "-X github.com/infomosquitos/ticketmosquito/internal/cliapp.Version=$(VERSION)" \
		-o ticketmosquito ./cmd/ticketmosquito

serve: build
	./ticketmosquito serve

test:
	go test ./...

vet:
	go vet ./...

build-extensions:
	cd chrome-extension && npm install && npm run build
	cd firefox-extension && npm install && npm run build

release-artifacts:
	./scripts/build-release.sh $(VERSION)

clean:
	rm -f ticketmosquito
	rm -rf dist
	go clean
	rm -rf chrome-extension/dist firefox-extension/dist

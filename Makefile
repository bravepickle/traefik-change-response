.PHONY: lint test bench vendor clean setup cover cover_web

export GO111MODULE=on

APP_SKIP_GOIMPORTS ?= 0
APP_SKIP_STATICCHECK ?= 0
OUT_TESTS_COVER ?= ./coverprofile.out

default: lint test

# format all GO files
# fmt: $(wildcard *.go */*.go)
fmt:
	go fmt ./...

ifeq ($(APP_SKIP_GOIMPORTS),1)
	@echo Skipping goimports...
else
	goimports -w .
endif

# static analysis (aka lint)
lint: fmt
	go vet ./...

ifeq ($(APP_SKIP_STATICCHECK),1)
	@echo Skipping staticcheck linting...
else
	staticcheck ./...
endif

test:
	go test -v -cover -race ./...

bench:
	go test -v -bench ./...

yaegi_test:
	yaegi test -v .

vendor:
	go mod vendor

clean:
	rm -rf ./vendor

setup:
	go mod tidy
	go mod vendor

ifneq ($(APP_SKIP_PKG_UPDATE),1)
	# update external apps
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/traefik/yaegi/cmd/yaegi@latest

	# update package dependencies
	go get -u ./...
endif

# run test coverage
cover: lint
	go test -coverprofile $(OUT_TESTS_COVER) ./...
	go tool cover -func $(OUT_TESTS_COVER)

# run test coverage in HTML view
cover_web: lint
	go test -coverprofile $(OUT_TESTS_COVER) ./...
	go tool cover -html $(OUT_TESTS_COVER)

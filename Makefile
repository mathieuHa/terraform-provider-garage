VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo dev)
TOFU ?= $(shell command -v tofu 2>/dev/null)

.PHONY: build install test testacc testacc-up testacc-down docs fmt vet lint ci clean

build:
	go build -o terraform-provider-garage

install: build
	@echo 'Add this to ~/.tofurc (or ~/.terraformrc) to use the local build:'
	@echo '  provider_installation {'
	@echo '    dev_overrides { "registry.opentofu.org/mathieuha/garage" = "$(CURDIR)" }'
	@echo '    direct {}'
	@echo '  }'

test:
	go test -race -count=1 ./...

testacc-up:
	scripts/testacc-garage.sh start > .testacc.env

testacc-down:
	scripts/testacc-garage.sh stop
	rm -f .testacc.env

testacc: testacc-up
	. ./.testacc.env && \
	TF_ACC=1 TF_ACC_TERRAFORM_PATH=$(TOFU) TF_ACC_PROVIDER_HOST=registry.opentofu.org TF_ACC_PROVIDER_NAMESPACE=hashicorp \
	go test -count=1 -v -timeout 20m -run '^TestAcc' ./internal/provider/

docs:
	scripts/docs.sh

fmt:
	gofmt -w .
	tofu fmt -recursive examples

vet:
	go vet ./...

lint:
	golangci-lint run ./...

ci: build vet lint test

clean:
	rm -f terraform-provider-garage .testacc.env

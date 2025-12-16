default: build

# Build the provider
build:
	go build -o terraform-provider-lakefs

# Install the provider locally for testing
install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/zjpiazza/lakefs/0.1.0/linux_amd64
	cp terraform-provider-lakefs ~/.terraform.d/plugins/registry.terraform.io/zjpiazza/lakefs/0.1.0/linux_amd64/

# Run unit tests
test:
	go test -v ./...

# Run acceptance tests
testacc:
	TF_ACC=1 go test ./internal/provider -v -timeout 120m

# Start test infrastructure
testacc-up:
	docker compose up -d
	@echo "Waiting for LakeFS to be healthy..."
	@sleep 5
	@WRITE_ENV_FILE=true ./scripts/setup-test-lakefs.sh

# Stop test infrastructure
testacc-down:
	docker compose down -v

# Run acceptance tests with local LakeFS (full workflow)
testacc-local: testacc-up
	@. ./.env.test && TF_ACC=1 go test ./internal/provider -v -timeout 120m

# Generate documentation
generate:
	go generate ./...

# Alias for backwards compatibility
docs: generate

# Lint the code
lint:
	golangci-lint run ./...

# Format the code
fmt:
	go fmt ./...
	terraform fmt -recursive ./examples/

# Clean build artifacts
clean:
	rm -f terraform-provider-lakefs
	rm -rf ~/.terraform.d/plugins/registry.terraform.io/zjpiazza/lakefs/

# Tidy dependencies
tidy:
	go mod tidy

# Run all checks before committing
check: fmt lint test

.PHONY: default build install test testacc testacc-up testacc-down testacc-local docs lint fmt clean tidy check

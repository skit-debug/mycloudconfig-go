GOLANGCI_LINT_VERSION := v2.12.2
GOMOCK_VERSION := v1.6.0

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

BIN_DIR := tooling/bin
MOCKGEN := $(BIN_DIR)/mockgen

GOLANGCI_LINT_VERSION_NUMBER := $(patsubst v%,%,$(GOLANGCI_LINT_VERSION))
ifeq ($(GOOS),windows)
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint.exe
GOLANGCI_LINT_ARCHIVE := golangci-lint-$(GOLANGCI_LINT_VERSION_NUMBER)-$(GOOS)-$(GOARCH).zip
else
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint
GOLANGCI_LINT_ARCHIVE := golangci-lint-$(GOLANGCI_LINT_VERSION_NUMBER)-$(GOOS)-$(GOARCH).tar.gz
endif
GOLANGCI_LINT_URL := https://github.com/golangci/golangci-lint/releases/download/$(GOLANGCI_LINT_VERSION)/$(GOLANGCI_LINT_ARCHIVE)
GOLANGCI_LINT_CHECKSUMS_URL := https://github.com/golangci/golangci-lint/releases/download/$(GOLANGCI_LINT_VERSION)/golangci-lint-$(GOLANGCI_LINT_VERSION_NUMBER)-checksums.txt

.PHONY: all tools lint test

tools: $(GOLANGCI_LINT) $(MOCKGEN)

$(GOLANGCI_LINT):
	@set -e; \
	tmp_dir=$$(mktemp -d); \
	trap 'rm -rf "$$tmp_dir"' EXIT; \
	echo "Downloading golangci-lint $(GOLANGCI_LINT_VERSION) for $(GOOS)/$(GOARCH)..."; \
	curl -fsSL "$(GOLANGCI_LINT_URL)" \
		-o "$$tmp_dir/$(GOLANGCI_LINT_ARCHIVE)"; \
	curl -fsSL "$(GOLANGCI_LINT_CHECKSUMS_URL)" \
		-o "$$tmp_dir/checksums.txt"; \
	cd "$$tmp_dir"; \
	grep "  $(GOLANGCI_LINT_ARCHIVE)$$" checksums.txt > checksum.txt; \
	if command -v sha256sum >/dev/null 2>&1; then \
		sha256sum -c checksum.txt; \
	elif command -v shasum >/dev/null 2>&1; then \
		shasum -a 256 -c checksum.txt; \
	else \
		echo "Error: sha256sum or shasum is required"; \
		exit 1; \
	fi; \
	if [ "$(GOOS)" = "windows" ]; then \
		unzip "$(GOLANGCI_LINT_ARCHIVE)" -d "$$tmp_dir"; \
	else \
		tar -xzf "$(GOLANGCI_LINT_ARCHIVE)" -C "$$tmp_dir"; \
	fi; \
	mkdir -p "$(CURDIR)/$(BIN_DIR)"; \
	if [ "$(GOOS)" = "windows" ]; then \
		cp "$$tmp_dir/golangci-lint-$(GOLANGCI_LINT_VERSION_NUMBER)-$(GOOS)-$(GOARCH)/golangci-lint.exe" \
			"$(CURDIR)/$(GOLANGCI_LINT)"; \
	else \
		cp "$$tmp_dir/golangci-lint-$(GOLANGCI_LINT_VERSION_NUMBER)-$(GOOS)-$(GOARCH)/golangci-lint" \
			"$(CURDIR)/$(GOLANGCI_LINT)"; \
		chmod +x "$(CURDIR)/$(GOLANGCI_LINT)"; \
	fi

$(MOCKGEN):
	@echo "Installing mockgen $(GOMOCK_VERSION)..."
	@mkdir -p "$(BIN_DIR)"
	@GOBIN="$(CURDIR)/$(BIN_DIR)" go install github.com/golang/mock/mockgen@$(GOMOCK_VERSION)

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

test:
	go test ./...

all: tools lint test
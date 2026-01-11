# Makefile for mkanban-tui

.PHONY: all build install clean test help arch-pkg wire

# Build variables
BINARY_TUI := mkanban
BINARY_DAEMON := mkanbad
BINARY_NOTES := mnotes
BINARY_AGENDA := magenda
PREFIX ?= /usr/local
BINDIR := $(PREFIX)/bin
SYSTEMD_USER_DIR := $(PREFIX)/lib/systemd/user
SYSTEMD_SYSTEM_DIR := $(PREFIX)/lib/systemd/system
COMPLETIONS_DIR := $(PREFIX)/share
DOCS_DIR := $(PREFIX)/share/doc/mkanban-tui

# Go build flags
GOFLAGS := -trimpath
LDFLAGS := -s -w

all: build ## Build both TUI client and daemon

build: ## Build all binaries (TUI, daemon, notes, agenda)
	@echo "Building TUI client..."
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_TUI) ./cmd/mkanban
	@echo "Building daemon..."
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_DAEMON) ./cmd/mkanbad
	@echo "Building notes CLI..."
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NOTES) ./cmd/mnotes
	@echo "Building agenda CLI..."
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_AGENDA) ./cmd/magenda
	@echo "Build complete!"

wire: ## Regenerate wire dependency injection
	@echo "Regenerating wire DI..."
	cd internal/di && wire
	@echo "Wire generation complete!"

install: build ## Install binaries, systemd services, and completions
	@echo "Installing binaries..."
	install -Dm755 $(BINARY_TUI) $(DESTDIR)$(BINDIR)/$(BINARY_TUI)
	install -Dm755 $(BINARY_DAEMON) $(DESTDIR)$(BINDIR)/$(BINARY_DAEMON)
	install -Dm755 $(BINARY_NOTES) $(DESTDIR)$(BINDIR)/$(BINARY_NOTES)
	install -Dm755 $(BINARY_AGENDA) $(DESTDIR)$(BINDIR)/$(BINARY_AGENDA)

	@echo "Installing systemd service files..."
	install -Dm644 systemd/mkanbad.service $(DESTDIR)$(SYSTEMD_USER_DIR)/mkanbad.service
	install -Dm644 systemd/mkanbad@.service $(DESTDIR)$(SYSTEMD_SYSTEM_DIR)/mkanbad@.service

	@echo "Generating and installing shell completions..."
	@mkdir -p $(DESTDIR)$(COMPLETIONS_DIR)/bash-completion/completions
	@mkdir -p $(DESTDIR)$(COMPLETIONS_DIR)/zsh/site-functions
	@mkdir -p $(DESTDIR)$(COMPLETIONS_DIR)/fish/vendor_completions.d
	./$(BINARY_TUI) completion bash > $(DESTDIR)$(COMPLETIONS_DIR)/bash-completion/completions/$(BINARY_TUI)
	./$(BINARY_TUI) completion zsh > $(DESTDIR)$(COMPLETIONS_DIR)/zsh/site-functions/_$(BINARY_TUI)
	./$(BINARY_TUI) completion fish > $(DESTDIR)$(COMPLETIONS_DIR)/fish/vendor_completions.d/$(BINARY_TUI).fish

	@echo "Installing documentation..."
	install -Dm644 README.md $(DESTDIR)$(DOCS_DIR)/README.md

	@echo "Installation complete!"

uninstall: ## Uninstall binaries, services, and completions
	@echo "Uninstalling..."
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY_TUI)
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY_DAEMON)
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY_NOTES)
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY_AGENDA)
	rm -f $(DESTDIR)$(SYSTEMD_USER_DIR)/mkanbad.service
	rm -f $(DESTDIR)$(SYSTEMD_SYSTEM_DIR)/mkanbad@.service
	rm -f $(DESTDIR)$(COMPLETIONS_DIR)/bash-completion/completions/$(BINARY_TUI)
	rm -f $(DESTDIR)$(COMPLETIONS_DIR)/zsh/site-functions/_$(BINARY_TUI)
	rm -f $(DESTDIR)$(COMPLETIONS_DIR)/fish/vendor_completions.d/$(BINARY_TUI).fish
	rm -rf $(DESTDIR)$(DOCS_DIR)
	@echo "Uninstallation complete!"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_TUI) $(BINARY_DAEMON) $(BINARY_NOTES) $(BINARY_AGENDA)
	rm -f *.bash *.zsh *.fish
	@echo "Clean complete!"

test: ## Run tests
	@echo "Running tests..."
	go test -v -race ./...

coverage: ## Generate test coverage report
	@echo "Generating coverage report..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	@echo "Format complete!"

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...
	@echo "Vet complete!"

lint: fmt vet ## Run formatters and linters
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

arch-pkg: ## Build Arch Linux package
	@echo "Building Arch Linux package..."
	makepkg -sf
	@echo "Package build complete!"

arch-install: arch-pkg ## Build and install Arch Linux package
	@echo "Installing Arch Linux package..."
	makepkg -sfi
	@echo "Package installed!"

systemd-user-enable: ## Enable user systemd service
	@echo "Enabling user systemd service..."
	systemctl --user daemon-reload
	systemctl --user enable mkanbad.service
	systemctl --user start mkanbad.service
	@echo "User service enabled and started!"

systemd-user-disable: ## Disable user systemd service
	@echo "Disabling user systemd service..."
	systemctl --user stop mkanbad.service
	systemctl --user disable mkanbad.service
	@echo "User service disabled and stopped!"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	@echo "Dependencies downloaded!"

help: ## Show this help message
	@echo "mkanban-tui Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

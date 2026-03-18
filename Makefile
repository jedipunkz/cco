BINARY_NAME := ax
INSTALL_DIR := $(HOME)/.bin
PID_FILE := $(HOME)/.ax/daemon.pid

.PHONY: build install reinstall clean

build:
	go build -o $(BINARY_NAME) .

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

reinstall: build
	@if [ -f $(PID_FILE) ]; then \
		PID=$$(cat $(PID_FILE)); \
		echo "Stopping daemon (PID=$$PID)..."; \
		kill $$PID 2>/dev/null || true; \
		sleep 1; \
	else \
		echo "No daemon PID file found, skipping kill."; \
	fi
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Reinstalled $(BINARY_NAME)."

clean:
	rm -f $(BINARY_NAME)

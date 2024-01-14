# Makefile for building and installing GitPilotAI

# The binary to build (just the basename).
BIN := gitpilotai

# Where to push the binary to, for install.
INSTALL_PATH := /usr/local/bin/

# Default target
all: build

# This will build the binary under the current directory.
build:
	@echo "Building $(BIN)..."
	@go build -o $(BIN)

# This will install the binary to INSTALL_PATH.
install:
	@echo "Installing $(BIN) to $(INSTALL_PATH)"
	@mv $(BIN) $(INSTALL_PATH)

# Phony targets
.PHONY: build install

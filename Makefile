RELEASE_MATRIX := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

CGO_ENABLED ?= 0
GOFLAGS     ?= -buildvcs=auto -trimpath
LDFLAGS     ?= -s -w
GOWORK      ?= off
GOFTAGS     ?= forceposix

NATIVE_GOOS      := $(shell go env GOOS)
NATIVE_GOARCH    := $(shell go env GOARCH)
NATIVE_EXTENSION := $(if $(filter $(NATIVE_GOOS),windows),.exe,)

BINARY     ?= yc-scheduler
PKG        ?= ./cmd/yc-scheduler
OUTPUT_DIR ?= build
GO         ?= go
LINTER     ?= golangci-lint
ALIGNER    ?= betteralign
CYCLONEDX  ?= cyclonedx-gomod

RACE ?= 0
ifeq ($(RACE),1)
	EXTRA_BUILD_FLAGS := -race
endif

MODULE  := $(shell go list -m)
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)
COMMIT  := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
URL     := https://$(MODULE)

LDFLAGS_X := \
	-X '$(MODULE)/internal/vars.Version=$(VERSION)' \
	-X '$(MODULE)/internal/vars.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/vars._buildTime=$(DATE)' \
	-X '$(MODULE)/internal/vars.URL=$(URL)'

.PHONY: all clean build release test cover lint vet tools align align-fix sbom sbom-app sbom-bin release-notes init check schema-gen

all: test build

init: tools
	@echo ">> downloading dependencies"
	$(GO) mod download
	@echo ">> initialization complete"

check: vet lint align test
	@echo ">> all checks passed"

clean:
	rm -rf $(OUTPUT_DIR)
	rm -f coverage.out

build: clean schema-gen
	@mkdir -p $(OUTPUT_DIR)
	@echo ">> building native: $(BINARY)$(NATIVE_EXTENSION)"
	GOOS=$(NATIVE_GOOS) GOARCH=$(NATIVE_GOARCH) \
	GOWORK=$(GOWORK) CGO_ENABLED=$(CGO_ENABLED) \
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS) $(LDFLAGS_X)" -tags "$(GOFTAGS)" $(EXTRA_BUILD_FLAGS) \
		-o $(OUTPUT_DIR)/$(BINARY)$(NATIVE_EXTENSION) $(PKG)

release: clean schema-gen
	@mkdir -p $(OUTPUT_DIR)
	@for target in $(RELEASE_MATRIX); do \
		goos=$${target%%/*}; \
		goarch=$${target##*/}; \
		ext=$$( [ $$goos = "windows" ] && echo ".exe" || echo "" ); \
		out="$(OUTPUT_DIR)/$(BINARY)-$${goos}-$${goarch}$$ext"; \
		echo ">> building $$out"; \
		GOOS=$$goos GOARCH=$$goarch \
		GOWORK=$(GOWORK) CGO_ENABLED=$(CGO_ENABLED) \
		$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS) $(LDFLAGS_X)" -tags "$(GOFTAGS)" $(EXTRA_BUILD_FLAGS) \
			-o $$out $(PKG); \
	done
	@$(MAKE) sbom-app

vet:
	@echo ">> running go vet"
	$(GO) vet ./...

test:
	@echo ">> running tests"
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -v -cover $(TEST_FLAGS) ./...

cover:
	@echo ">> running tests with coverage"
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

tools:
	@echo ">> installing golangci-lint"
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@echo ">> installing betteralign"
	$(GO) install github.com/orijtech/betteralign/cmd/betteralign@latest
	@echo ">> installing cyclonedx-gomod"
	$(GO) install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest

lint:
	@echo ">> running golangci-lint"
	$(LINTER) run ./...

align:
	$(ALIGNER) ./...

align-fix:
	$(ALIGNER) -apply ./...

sbom: sbom-app sbom-bin

sbom-app:
	@echo ">> SBOM (app)"
	$(CYCLONEDX) app -json -packages -files -licenses \
		-output "$(OUTPUT_DIR)/$(BINARY).sbom.json" -main "$(PKG)"

sbom-bin:
	@echo ">> SBOM (bin native if exists)"
	@[ -f "$(OUTPUT_DIR)/$(BINARY)$(NATIVE_EXTENSION)" ] && \
		$(CYCLONEDX) bin -json -output "$(OUTPUT_DIR)/$(BINARY)$(NATIVE_EXTENSION).sbom.json" \
			"$(OUTPUT_DIR)/$(BINARY)$(NATIVE_EXTENSION)" || true

release-notes:
	@awk '\
	/^<!--/,/^-->/ { next } \
	/^## \[[0-9]+\.[0-9]+\.[0-9]+\]/ { if (found) exit; found=1; next } found { print } \
	' CHANGELOG.md

schema-gen:
	@echo ">> generating JSON schema"
	@mkdir -p static/schemas
	$(GO) run ./cmd/schema-gen -out static/schemas/config.json

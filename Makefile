# rcplug-plugins: plugins for rcplug on the RØDECaster. Every plugin is a
# directory <name>/ with plugin.json and a Go main package. Pure Go standard
# library plus the rcplug SDK. Run "make help".

GO       ?= go
VERSION  := $(shell cat VERSION)
DIST     := dist
HOSTARCH := $(shell $(GO) env GOARCH)
HOSTOS   := $(shell $(GO) env GOOS)
# A plugin's binary reports its own version (from its plugin.json) and the
# release of this repository it came with.
LDFLAGS  := -s -w -X main.release=$(VERSION)
GOBUILD   = CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) $(GO) build -trimpath

PLUGINS  := $(notdir $(patsubst %/,%,$(dir $(wildcard */plugin.json))))

# The rcplug platform repository, for run-local.
RCPLUG ?= ../rcplug
# Signs releases; the version go.mod pins (in a go.work: the local clone).
RELEASE_TOOL ?= $(GO) run github.com/Hustenreizjuengling/rcplug/cmd/rcplug-release

.PHONY: help all check fmt fmt-check vet test test-race test-arm64 build build-arm64 build-host \
        package sign run-local clean docs-check check-versions

help: ## show this help
	@grep -hE '^[a-z0-9-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-15s %s\n", $$1, $$2}'

all: check package ## run all checks and build the plugin packages

check: fmt-check vet test-race docs-check ## everything CI runs (except arm64)

fmt: ## format the Go code
	gofmt -w .

fmt-check: ## fail if Go code is not formatted
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "not gofmt'ed:"; echo "$$out"; exit 1; fi

vet: ## go vet
	$(GO) vet ./...

check-versions: ## every plugin that changed since the last release has a new version in its plugin.json
	tools/check-plugin-versions.sh

docs-check: ## every relative link and #anchor in the Markdown files resolves
	$(GO) run ./tools/doclinks

test: ## unit tests
	$(GO) test ./...

test-race: ## unit tests with the race detector
	$(GO) test -race ./...

test-arm64: ## unit tests as arm64 binaries under qemu-user (needs qemu-aarch64)
	CGO_ENABLED=0 GOARCH=arm64 $(GO) test -exec qemu-aarch64 ./...

build: build-arm64 build-host ## build every plugin for the device and this machine

build-arm64:
	@$(MAKE) --no-print-directory bin OS=linux ARCH=arm64

build-host:
	@$(MAKE) --no-print-directory bin OS=$(HOSTOS) ARCH=$(HOSTARCH)

.PHONY: bin
bin:
	@for p in $(PLUGINS); do echo "  $(OS)/$(ARCH) $$p"; mkdir -p $(DIST)/$(OS)-$(ARCH)/$$p; \
	  v="$$(sed -n 's/^  "version": "\([^"]*\)",$$/\1/p' $$p/plugin.json)"; \
	  $(GOBUILD) -ldflags "$(LDFLAGS) -X main.version=$$v" -o $(DIST)/$(OS)-$(ARCH)/$$p/$$p ./$$p || exit 1; \
	  cp $$p/plugin.json $(DIST)/$(OS)-$(ARCH)/$$p/; done

package: build-arm64 ## one tarball per plugin in dist/ (<name>-<version>-aarch64.tar.gz) and SHA256SUMS
	@tmp="$$(mktemp -d)"; trap 'rm -rf "$$tmp"' EXIT; set -e; \
	for p in $(PLUGINS); do mkdir -p "$$tmp/$$p"; cp $(DIST)/linux-arm64/$$p/$$p $$p/plugin.json "$$tmp/$$p/"; \
	  chmod 0644 "$$tmp/$$p/plugin.json"; chmod 0755 "$$tmp/$$p/$$p"; \
	  tar --owner=0 --group=0 -C "$$tmp/$$p" -czf $(DIST)/$$p-$(VERSION)-aarch64.tar.gz .; done; \
	cd $(DIST) && sha256sum $(foreach p,$(PLUGINS),$(p)-$(VERSION)-aarch64.tar.gz) > SHA256SUMS
	@echo "packaged: $(foreach p,$(PLUGINS),$(DIST)/$(p)-$(VERSION)-aarch64.tar.gz) (+ SHA256SUMS)"

sign: ## sign the packages in dist/ (release.json + .sig, checked by rcplugd before updates); key from $$RCPLUG_SIGNING_KEY
	$(RELEASE_TOOL) sign --project rcplug-plugins --version $(VERSION) --out $(DIST) \
	  $(foreach p,$(PLUGINS),$(DIST)/$(p)-$(VERSION)-aarch64.tar.gz)

run-local: build-host ## run the rcplug host (from $(RCPLUG)) with these plugins against the simulator
	$(MAKE) -C $(RCPLUG) run-local EXTRA_PLUGINS="$(foreach p,$(PLUGINS),$(abspath $(DIST)/$(HOSTOS)-$(HOSTARCH)/$(p)))"

clean: ## remove dist/
	rm -rf $(DIST)

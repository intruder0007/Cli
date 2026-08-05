EXE := $(shell go env GOEXE)

# templates/registry.txt is the single source of truth for which
# template plugins exist — adding a language means adding a directory
# under templates/ and one line here (see docs/templates/authoring.md).
# plugins/builtin (capabilities) stays a fixed, hand-maintained list:
# unlike templates, it doesn't grow with the language rollout.
TEMPLATES := $(shell cat templates/registry.txt)
CAPABILITIES := git-init readme github-actions-ci

# go.work workspaces don't expand "./..." from the repo root (it isn't a
# module itself), so every module must be listed explicitly here.
MODULES := ./cli/... ./core/... ./sdk/go/... \
	$(foreach t,$(TEMPLATES),./templates/$(t)/...) \
	$(foreach c,$(CAPABILITIES),./plugins/builtin/$(c)/...) \
	./tests/...

# See ADR-0006: version is injected via ldflags rather than hand-edited.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

EMBED_DIR := cli/internal/embedded/assets

.PHONY: build stage-embedded test vet run clean

# Plugins must build (and be staged into $(EMBED_DIR) for go:embed) before
# `cli` builds — see ADR-0012's "Build-order coupling" — so `cli` embeds
# its own platform's plugin binaries as a last-resort fallback for
# distribution methods (e.g. `go install`) that produce only the one
# binary, no sibling directories. Each template/capability's binary is
# expected to be named identically to its directory.
build: stage-embedded
	go build -ldflags "$(LDFLAGS)" -o bin/lumo$(EXE) ./cli

stage-embedded:
	rm -rf $(EMBED_DIR)/templates $(EMBED_DIR)/plugins
	for t in $(TEMPLATES); do \
		go build -o templates/$$t/$$t$(EXE) ./templates/$$t && \
		mkdir -p $(EMBED_DIR)/templates/$$t && \
		cp templates/$$t/plugin.json templates/$$t/$$t$(EXE) $(EMBED_DIR)/templates/$$t/ || exit 1; \
	done
	for c in $(CAPABILITIES); do \
		go build -o plugins/builtin/$$c/$$c$(EXE) ./plugins/builtin/$$c && \
		mkdir -p $(EMBED_DIR)/plugins/builtin/$$c && \
		cp plugins/builtin/$$c/plugin.json plugins/builtin/$$c/$$c$(EXE) $(EMBED_DIR)/plugins/builtin/$$c/ || exit 1; \
	done

test:
	go test $(MODULES)

vet:
	go vet $(MODULES)

run: build
	./bin/lumo$(EXE) new

clean:
	rm -f bin/lumo$(EXE)
	for t in $(TEMPLATES); do rm -f templates/$$t/$$t$(EXE); done
	for c in $(CAPABILITIES); do rm -f plugins/builtin/$$c/$$c$(EXE); done
	rm -rf $(EMBED_DIR)/templates $(EMBED_DIR)/plugins

EXE := $(shell go env GOEXE)

# go.work workspaces don't expand "./..." from the repo root (it isn't a
# module itself), so every module must be listed explicitly here.
MODULES := ./cli/... ./core/... ./sdk/go/... ./templates/go-rest-api/... ./templates/node-rest-api/... \
	./plugins/builtin/git-init/... ./plugins/builtin/readme/... ./plugins/builtin/github-actions-ci/... ./tests/...

# See ADR-0006: version is injected via ldflags rather than hand-edited.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

EMBED_DIR := cli/internal/embedded/assets

.PHONY: build stage-embedded test vet run clean

# Plugins must build (and be staged into $(EMBED_DIR) for go:embed) before
# `cli` builds — see ADR-0012's "Build-order coupling" — so `cli` embeds
# its own platform's plugin binaries as a last-resort fallback for
# distribution methods (e.g. `go install`) that produce only the one
# binary, no sibling directories.
build: stage-embedded
	go build -ldflags "$(LDFLAGS)" -o bin/lumo$(EXE) ./cli

stage-embedded:
	go build -o templates/go-rest-api/go-rest-api$(EXE) ./templates/go-rest-api
	go build -o templates/node-rest-api/node-rest-api$(EXE) ./templates/node-rest-api
	go build -o plugins/builtin/git-init/git-init$(EXE) ./plugins/builtin/git-init
	go build -o plugins/builtin/readme/readme$(EXE) ./plugins/builtin/readme
	go build -o plugins/builtin/github-actions-ci/github-actions-ci$(EXE) ./plugins/builtin/github-actions-ci
	rm -rf $(EMBED_DIR)/templates $(EMBED_DIR)/plugins
	mkdir -p $(EMBED_DIR)/templates/go-rest-api $(EMBED_DIR)/templates/node-rest-api \
		$(EMBED_DIR)/plugins/builtin/git-init $(EMBED_DIR)/plugins/builtin/readme $(EMBED_DIR)/plugins/builtin/github-actions-ci
	cp templates/go-rest-api/plugin.json templates/go-rest-api/go-rest-api$(EXE) $(EMBED_DIR)/templates/go-rest-api/
	cp templates/node-rest-api/plugin.json templates/node-rest-api/node-rest-api$(EXE) $(EMBED_DIR)/templates/node-rest-api/
	cp plugins/builtin/git-init/plugin.json plugins/builtin/git-init/git-init$(EXE) $(EMBED_DIR)/plugins/builtin/git-init/
	cp plugins/builtin/readme/plugin.json plugins/builtin/readme/readme$(EXE) $(EMBED_DIR)/plugins/builtin/readme/
	cp plugins/builtin/github-actions-ci/plugin.json plugins/builtin/github-actions-ci/github-actions-ci$(EXE) $(EMBED_DIR)/plugins/builtin/github-actions-ci/

test:
	go test $(MODULES)

vet:
	go vet $(MODULES)

run: build
	./bin/lumo$(EXE) new

clean:
	rm -f bin/lumo$(EXE)
	rm -f templates/go-rest-api/go-rest-api$(EXE)
	rm -f templates/node-rest-api/node-rest-api$(EXE)
	rm -f plugins/builtin/git-init/git-init$(EXE)
	rm -f plugins/builtin/readme/readme$(EXE)
	rm -f plugins/builtin/github-actions-ci/github-actions-ci$(EXE)
	rm -rf $(EMBED_DIR)/templates $(EMBED_DIR)/plugins

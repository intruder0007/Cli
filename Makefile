EXE := $(shell go env GOEXE)

# go.work workspaces don't expand "./..." from the repo root (it isn't a
# module itself), so every module must be listed explicitly here.
MODULES := ./cli/... ./core/... ./sdk/go/... ./templates/go-rest-api/... \
	./plugins/builtin/git-init/... ./plugins/builtin/readme/... ./plugins/builtin/github-actions-ci/...

.PHONY: build test vet run clean

build:
	go build -o bin/bootstrap$(EXE) ./cli
	go build -o templates/go-rest-api/go-rest-api$(EXE) ./templates/go-rest-api
	go build -o plugins/builtin/git-init/git-init$(EXE) ./plugins/builtin/git-init
	go build -o plugins/builtin/readme/readme$(EXE) ./plugins/builtin/readme
	go build -o plugins/builtin/github-actions-ci/github-actions-ci$(EXE) ./plugins/builtin/github-actions-ci

test:
	go test $(MODULES)

vet:
	go vet $(MODULES)

run: build
	./bin/bootstrap$(EXE) new

clean:
	rm -f bin/bootstrap$(EXE)
	rm -f templates/go-rest-api/go-rest-api$(EXE)
	rm -f plugins/builtin/git-init/git-init$(EXE)
	rm -f plugins/builtin/readme/readme$(EXE)
	rm -f plugins/builtin/github-actions-ci/github-actions-ci$(EXE)

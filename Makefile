# Ajayeb task runner. `make` (or `make help`) lists the targets.
#
# Every target that compiles, tests, builds, lints, installs or starts a
# browser runs its command through tools/heavy.py, the machine-wide memory,
# slot and run-mode guard (docs/dev/ram-policy.md, `python3 tools/heavy.py
# --help`). When the guard refuses or gives up waiting, the step exits 75 and
# prints a line starting with "heavy: refused"; make then reports "Error 75"
# and exits 2 itself. `python3 tools/heavy.py status` shows why.
#
# The aggregate targets (verify-quick, verify, verify-full) are not guarded
# as a whole: each step takes and releases its own slot, so other agents can
# take turns between steps. To hold one slot for a whole run instead, wrap it:
# `python3 tools/heavy.py run --kind test -- make verify-quick`. Guarded steps
# inside it then pass straight through (AJAYEB_HEAVY_HELD=1), except that a
# browser step still takes the browser lock.
#
# .NOTPARALLEL keeps the steps in order, one at a time, even under make -j.
# Written for GNU Make 3.81, the version macOS ships as /usr/bin/make.

SHELL := /bin/sh
.DEFAULT_GOAL := help
.NOTPARALLEL:
.SUFFIXES:

HEAVY := python3 "$(CURDIR)/tools/heavy.py"

# Knobs. Set them on the command line, for example `make e2e GREP=@full`.
GREP ?=
PW_ARGS ?=
E2E_BUILD ?= 1
PROD_URL ?= https://ajayeb.lilf.ir
AJAYEB_SWEEP ?= 300
GO_PKGS ?= ./...
GO_RUN ?=
TEST_TIMEOUT ?= 20m
FUZZTIME ?= 5m
FUZZ ?= FuzzPlayout
FUZZ_PKG ?= ./internal/engine
BENCH_ARGS ?=
GEN_ARGS ?=
VERSION = $(shell git describe --always --dirty 2>/dev/null || echo dev)

RACE_DIRS := internal/room internal/ws
PNPM_FROZEN := $(if $(wildcard pnpm-lock.yaml),--frozen-lockfile)
E2E_PREREQS := $(if $(filter 1,$(E2E_BUILD)),build-server build-web-test)

.PHONY: help install install-offline dev gen gen-check gen-check-if-ready \
        fmt-check vet test-go test-race lint-go build-server fuzz bench \
        build-web build-web-test test-web check-web lint-web build \
        e2e e2e-smoke e2e-prod shots vec-lint vec-shot vec-blind vec-family \
        test-py test-tools verify-quick verify verify-full clean

##@ Getting started

help: ## List the targets and the variables they read (the default target)
	@printf 'Usage: make <target> [VAR=value ...]\n'
	@awk '/^##@ / { printf "\n%s\n", substr($$0, 5); next } /^[a-zA-Z0-9_-]+:.*## / { name = $$0; sub(/:.*/, "", name); desc = $$0; sub(/^.*## /, "", desc); printf "  %-17s %s\n", name, desc }' $(MAKEFILE_LIST)
	@printf '\nVariables (set on the command line, e.g. make e2e GREP=@full):\n'
	@printf '  %-19s %s\n' \
	  'GREP=@full' 'Playwright --grep filter for e2e' \
	  'PW_ARGS=...' 'extra Playwright flags for e2e and shots' \
	  'E2E_BUILD=0' 'skip the server and test-mode web builds before e2e and shots' \
	  'PROD_URL=...' 'site for e2e-prod (default https://ajayeb.lilf.ir)' \
	  'AJAYEB_SWEEP=300' 'games in the Go playout sweep (verify-full uses 5000)' \
	  'GO_PKGS=./...' 'packages for test-go, e.g. ./internal/engine/...' \
	  'GO_RUN=regexp' 'go test -run filter for test-go' \
	  'TEST_TIMEOUT=20m' 'go test -timeout for test-go and test-race' \
	  'FUZZTIME=5m' 'how long fuzz runs' \
	  'FUZZ=FuzzPlayout' 'the fuzz target for fuzz, e.g. FuzzDecodeClient' \
	  'FUZZ_PKG=./internal/engine' 'its package, e.g. ./internal/wire' \
	  'BENCH_ARGS=...' 'extra flags for ajayeb-sim bench' \
  'SET=chain' 'the icon set (art/vector/sets.json) for vec-shot and vec-blind' \
  'ITER=2' 'vec-shot writes into <set>/iter-N/ to keep iterations apart' \
  'SIZES=12@2,16@1' 'vec-blind sheet sizes as px@dpr' \
  'READER=<dir>' 'vec-blind also copies the sheets and names, never the key, to <dir>' \
	  'GEN_ARGS=...' 'extra flags for ajayeb-gen' \
	  'VERSION=...' 'version stamped into the release build (default: git describe)'
	@printf '\nA step that fails with "Error 75" was refused by tools/heavy.py; run python3 tools/heavy.py status.\n'

install: ## Download the Go modules, pnpm install (--frozen-lockfile once pnpm-lock.yaml exists) and Playwright's Chromium
	$(HEAVY) run --kind install -- go -C server mod download
	$(HEAVY) run --kind install -- pnpm install $(PNPM_FROZEN)
	$(HEAVY) run --kind install -- pnpm -C e2e exec playwright install chromium --no-shell

install-offline: ## Install from the local caches only, for worktrees (GOPROXY=off, pnpm --offline)
	GOPROXY=off $(HEAVY) run --kind install -- go -C server mod download
	$(HEAVY) run --kind install -- pnpm install --offline --frozen-lockfile

dev: ## Explain how to run the dev servers with self_host.py dev-start (starts nothing itself)
	@printf '%s\n' \
	  'make dev starts nothing by itself. The dev servers run in tmux under self_host.py,' \
	  'so they outlive this shell and can be stopped cleanly:' \
	  '' \
	  '  python3 self_host.py dev-start   # Go server (go tool air) on 127.0.0.1:38951, Vite on port 38952' \
	  '  python3 self_host.py --help      # the other subcommands' \
	  '' \
	  'Never start dev servers in a worktree or during unattended work (see AGENTS.md).'

##@ Code generation

gen: ## Regenerate web/src/lib/wire: tygo for the wire types, then ajayeb-gen
	$(HEAVY) run --kind build -- go -C server tool tygo generate
ifneq ($(wildcard server/cmd/ajayeb-gen),)
	$(HEAVY) run --kind build -- go -C server run ./cmd/ajayeb-gen $(GEN_ARGS)
else
	@echo "gen: server/cmd/ajayeb-gen does not exist yet, so only tygo ran"
endif

# gen-check compares against a snapshot of the working tree, not against git,
# so it passes on uncommitted but regenerated files (an agent's finished
# slice) and, on a clean checkout such as CI, it checks the committed files.
gen-check: ## Run gen and fail if that changed anything under web/src/lib/wire (the new files stay)
	@rm -rf .cache/gen-check && mkdir -p .cache/gen-check/before
	@if [ -d web/src/lib/wire ]; then cp -R web/src/lib/wire/. .cache/gen-check/before/; fi
	@$(MAKE) --no-print-directory gen
	@if diff -r .cache/gen-check/before web/src/lib/wire >/dev/null 2>&1; then \
	  rm -rf .cache/gen-check; echo "gen-check: web/src/lib/wire is up to date"; \
	else \
	  echo "gen-check: make gen changed web/src/lib/wire; keep the regenerated files:" >&2; \
	  diff -ru .cache/gen-check/before web/src/lib/wire >&2; \
	  rm -rf .cache/gen-check; exit 1; \
	fi

gen-check-if-ready:
	@if [ -d server/cmd/ajayeb-gen ]; then $(MAKE) --no-print-directory gen-check; \
	else echo "verify-quick: skipping gen-check until server/cmd/ajayeb-gen exists"; fi

##@ Go server (server/)

fmt-check: ## Fail if gofmt would change any Go file under server/
	@out=$$(gofmt -l server) || exit 1; \
	if [ -n "$$out" ]; then echo "fmt-check: these files need gofmt -w:" >&2; echo "$$out" >&2; exit 1; fi

vet: ## go vet ./... in server/
	$(HEAVY) run --kind lint -- go -C server vet ./...

test-go: ## Go tests with -shuffle=on (AJAYEB_SWEEP games in the playout sweep, default 300)
	AJAYEB_SWEEP=$(AJAYEB_SWEEP) $(HEAVY) run --kind test -- \
	  go -C server test -shuffle=on -timeout $(TEST_TIMEOUT) $(if $(GO_RUN),-run '$(GO_RUN)') $(GO_PKGS)

test-race: ## go test -race on internal/room and internal/ws (skipped while they have no tests)
	@pkgs=; for d in $(RACE_DIRS); do \
	  if [ -n "$$(find server/$$d -name '*_test.go' 2>/dev/null | head -n 1)" ]; then pkgs="$$pkgs ./$$d/..."; fi; \
	done; \
	if [ -z "$$pkgs" ]; then echo "test-race: no tests under $(RACE_DIRS) yet, skipping"; exit 0; fi; \
	echo "go -C server test -race$$pkgs"; \
	AJAYEB_SWEEP=$(AJAYEB_SWEEP) $(HEAVY) run --kind test -- go -C server test -race -timeout $(TEST_TIMEOUT) $$pkgs

lint-go: ## golangci-lint run in server/
	cd server && $(HEAVY) run --kind lint -- golangci-lint run

build-server: ## The Go server without the embedded web build, at server/bin/ajayeb
	$(HEAVY) run --kind build -- go -C server build -o bin/ajayeb ./cmd/ajayeb

fuzz: ## Fuzz FUZZ (default FuzzPlayout) in FUZZ_PKG (default ./internal/engine) for FUZZTIME (default 5m), under nice 10
	$(HEAVY) run --kind fuzz -- go -C server test -run '^$$' -fuzz '^$(FUZZ)$$' -fuzztime $(FUZZTIME) -parallel 2 $(FUZZ_PKG)

bench: ## The AI tournament bench (ajayeb-sim bench), under nice 10
	$(HEAVY) run --kind bench -- go -C server run ./cmd/ajayeb-sim bench $(BENCH_ARGS)

##@ Web client (web/)

build-web: ## Production SPA build into web/build
	$(HEAVY) run --kind build -- pnpm -C web build

build-web-test: ## Test-mode SPA build (vite --mode test) into web/build, for e2e and screenshots
	$(HEAVY) run --kind build -- pnpm -C web build --mode test

test-web: ## Web unit tests (pnpm -C web test, vitest)
	$(HEAVY) run --kind test -- pnpm -C web test

check-web: ## Type checks: svelte-check on web (including the recorded wire fixtures), tsc on e2e
	$(HEAVY) run --kind check -- pnpm -C web check
	$(HEAVY) run --kind check -- pnpm -C e2e check

lint-web: ## Lint and format checks for web and e2e (pnpm -C web lint, pnpm -C e2e lint)
	$(HEAVY) run --kind lint -- pnpm -C web lint
	$(HEAVY) run --kind lint -- pnpm -C e2e lint

##@ Release build

build: build-web ## Release binary with the web build embedded (-tags embedweb), at server/bin/ajayeb
	rsync -a --delete --exclude .gitkeep web/build/ server/internal/webdist/dist/
	$(HEAVY) run --kind build -- go -C server build -tags embedweb -trimpath \
	  -ldflags "-s -w -X main.version=$(VERSION)" -o bin/ajayeb ./cmd/ajayeb

##@ Browser runs (one headless Chrome on the machine at a time)

e2e: $(E2E_PREREQS) ## Playwright specs (GREP filters, PW_ARGS adds flags); builds first unless E2E_BUILD=0
	$(HEAVY) run --kind browser -- pnpm -C e2e exec playwright test $(if $(GREP),--grep '$(GREP)') $(PW_ARGS)

e2e-smoke: ## The @smoke e2e specs
	$(MAKE) --no-print-directory e2e GREP=@smoke

e2e-prod: ## The production smoke (@prod specs) against PROD_URL; no local build or server (docs/testing.md)
	E2E_BASE_URL=$(PROD_URL) $(HEAVY) run --kind browser -- pnpm -C e2e exec playwright test $(PW_ARGS)

# The ux project (e2e/ux/) captures the M11 review loop's screenshot set with
# its probes and budgets; docs/testing.md "UX screenshot matrix and probes".
shots: $(if $(wildcard e2e/ux),$(E2E_PREREQS)) ## The UX screenshot matrix with probes and budgets (Playwright project ux; SHOTS_DIR, PW_ARGS='--grep <screen>'); builds first unless E2E_BUILD=0
ifneq ($(wildcard e2e/ux),)
	$(HEAVY) run --kind browser -- pnpm -C e2e exec playwright test --project ux $(PW_ARGS)
else
	@echo "shots: e2e/ux/ is missing, skipping"
endif

##@ Vector art (tools/art/vector; outputs go to the run dir, never the repo)

vec-lint: ## Lint every icon SVG against art/vector/STYLE.md, names.ts and the sprite budget
	$(HEAVY) run --kind light -- node tools/art/vector/lint.mjs

vec-shot: ## Specimen shots of SET (light, dark, greyscale, colour-vision) and the pixel peep; ITER=n keeps iterations apart
	@test -n "$(SET)" || { echo "vec-shot: set SET=<name> (art/vector/sets.json)"; exit 2; }
	$(HEAVY) run --kind browser -- node tools/art/vector/shoot.mjs --set '$(SET)' $(if $(ITER),--iter '$(ITER)')
	node tools/art/vector/peep.mjs --set '$(SET)' $(if $(ITER),--iter '$(ITER)')

vec-blind: ## Shuffled, unlabelled blind-test sheets of SET (SIZES=12@2,14@2,16@1, READER=<dir> for a key-free copy); the key stays in the run dir
	@test -n "$(SET)" || { echo "vec-blind: set SET=<name> (art/vector/sets.json)"; exit 2; }
	$(HEAVY) run --kind browser -- node tools/art/vector/blind.mjs make --set '$(SET)' $(if $(SIZES),--sizes '$(SIZES)') $(if $(READER),--reader-dir '$(READER)')

vec-family: ## Every drawn icon of every set at 16 and 24 px, on paper and night, for cross-set review
	$(HEAVY) run --kind browser -- node tools/art/vector/family.mjs

##@ Verification

test-py: ## Python unit tests for tools/heavy.py and self_host.py (guard kind light)
	$(HEAVY) run --kind light -- python3 -m unittest discover -s tests

test-tools: ## Browser-free screenshot tool CLI argument and diagnostic tests
	$(HEAVY) run --kind light -- node --test tools/chrome/cli.test.mjs

verify-quick: fmt-check vet gen-check-if-ready test-go lint-web check-web test-web test-py test-tools vec-lint ## gofmt, go vet, gen-check, Go tests (sweep 300), web lint, svelte-check, vitest, Python tests, tool CLI tests, icon lint

verify: verify-quick lint-go test-race build e2e-smoke ## verify-quick, then golangci-lint, -race on room and ws, the release build and the e2e smoke

verify-full: verify ## verify, then a 5000-game sweep, fuzzing, the AI bench, every e2e spec and the screenshot matrix
	$(MAKE) --no-print-directory test-go AJAYEB_SWEEP=5000 TEST_TIMEOUT=90m
	$(MAKE) --no-print-directory fuzz bench e2e shots

##@ Housekeeping

clean: ## Remove build outputs: server/bin, web/build, .cache and the web build copied into webdist
	rm -rf server/bin web/build .cache
	@if [ -d server/internal/webdist/dist ]; then \
	  find server/internal/webdist/dist -mindepth 1 -maxdepth 1 ! -name .gitkeep -exec rm -rf {} +; fi

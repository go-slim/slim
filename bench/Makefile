# Makefile for standardized Go HTTP framework benchmarks
# Usage examples:
#   make bench           # short, stable run
#   make bench-long      # longer, more stable run
#   make chart           # generate chart from latest runs
#   make compare BASE=out/prev.txt NEW=out/run_slim.txt
#   make all             # bench + chart
#
# Requirements:
#   - Go toolchain
#   - benchstat (installed by `make deps`)

SHELL := /bin/bash
BENCH_PKGS := ./...
OUT := out
COUNT := 3
BENCHTIME := 200ms
CPU := 1

# Default list of framework filters (regex). Do not include quotes here.
F_SLIM  := .*_Slim$
F_GIN   := .*_Gin$
F_ECHO  := .*_Echo$
F_FIBER := .*_Fiber$
F_CHI   := .*_Chi$

.PHONY: all bench bench-long chart compare deps env clean

all: bench chart

bench:
	@mkdir -p $(OUT)
	@echo "[bench] Running Slim..."
	@GOWORK=off GOMAXPROCS=$(CPU) go test -run=^$$ -bench '$(F_SLIM)'  -benchmem -count=$(COUNT) -benchtime=$(BENCHTIME) $(BENCH_PKGS) > $(OUT)/run_slim.txt
	@echo "[bench] Running Gin..."
	@GOWORK=off GOMAXPROCS=$(CPU) go test -run=^$$ -bench '$(F_GIN)'   -benchmem -count=$(COUNT) -benchtime=$(BENCHTIME) $(BENCH_PKGS) > $(OUT)/run_gin.txt
	@echo "[bench] Running Echo..."
	@GOWORK=off GOMAXPROCS=$(CPU) go test -run=^$$ -bench '$(F_ECHO)'  -benchmem -count=$(COUNT) -benchtime=$(BENCHTIME) $(BENCH_PKGS) > $(OUT)/run_echo.txt
	@echo "[bench] Running Fiber..."
	@GOWORK=off GOMAXPROCS=$(CPU) go test -run=^$$ -bench '$(F_FIBER)' -benchmem -count=$(COUNT) -benchtime=$(BENCHTIME) $(BENCH_PKGS) > $(OUT)/run_fiber.txt
	@echo "[bench] Running Chi..."
	@GOWORK=off GOMAXPROCS=$(CPU) go test -run=^$$ -bench '$(F_CHI)'   -benchmem -count=$(COUNT) -benchtime=$(BENCHTIME) $(BENCH_PKGS) > $(OUT)/run_chi.txt
	@echo "[bench] Done. Outputs in $(OUT)/run_*.txt"

bench-long: COUNT=5
bench-long: BENCHTIME=1s
bench-long: bench

chart:
	@GOWORK=off go run ./cmd/chart
	@echo "[chart] wrote $(OUT)/chart.html"

# Compare two runs using benchstat
# Usage: make compare BASE=out/old.txt NEW=out/new.txt
compare:
	@if ! command -v benchstat >/dev/null 2>&1; then echo "benchstat not found. Run: make deps"; exit 1; fi
	@if [ -z "$(BASE)" ] || [ -z "$(NEW)" ]; then echo "Usage: make compare BASE=out/old.txt NEW=out/new.txt"; exit 2; fi
	@benchstat $(BASE) $(NEW)

deps:
	@echo "[deps] Installing benchstat..."
	@GOWORK=off go install golang.org/x/perf/cmd/benchstat@latest
	@echo "[deps] Done. Ensure \"$(shell go env GOPATH)/bin\" is in your PATH."

env:
	@echo "Go version:" && go version
	@echo "GOMAXPROCS=$(CPU)" && echo "COUNT=$(COUNT) BENCHTIME=$(BENCHTIME)"
	@echo "CPU info (macOS):" && sysctl -n machdep.cpu.brand_string || true

clean:
	@rm -f $(OUT)/run_*.txt $(OUT)/chart.html
	@echo "[clean] cleaned $(OUT)"

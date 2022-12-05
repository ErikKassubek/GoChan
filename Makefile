IN?=./main


all: build_instrumenter run_instrumenter build_prog

build_instrumenter:
	@cd instrumenter; go build

run_instrumenter:
	@echo ============= Instrument files =============
	@./instrumenter/instrumenter -in="$(IN)" -chan -mut -show_trace

build_prog:
	@cd output; cd $(IN); echo ============= Install libraries =============; go get github.com/ErikKassubek/GoChan/goChan@4a5420a; go get golang.org/x/tools/cmd/goimports; go mod tidy ; echo ============= Cleanup files ============= ;goimports -w . ; echo ============= Build files =============; go build; echo ============= Done =============
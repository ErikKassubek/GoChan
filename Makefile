IN?=./main
EXEC?=main


all: build_instrumenter run_instrumenter build_prog run

build_instrumenter:
	@cd instrumenter; go build

run_instrumenter:
	@echo ============= Instrument files =============
	@./instrumenter/instrumenter -in="$(IN)"
	@cd output; go build ./main.go

build_prog:
	@cd output; cd $(IN); echo ============= Install libraries =============; go get github.com/ErikKassubek/GoChan/goChan@4741dc6; go get golang.org/x/tools/cmd/goimports; go mod tidy ; echo ============= Cleanup files ============= ;goimports -w . ; echo ============= Build files =============; go build;

run:
	@echo 
	@echo ============= Start Analysis =============
	@cd output; ./main

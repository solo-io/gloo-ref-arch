.PHONY: gen-docs
gen-docs:
	go run scripts/gen-docs/main.go

.PHONY: run-all
run-all:
	go run scripts/run-all/main.go
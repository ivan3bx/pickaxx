VERSION     ?= $(shell git log -n 1 --format=%h)
GIT_HASH    := $(shell echo $(VERSION) | cut -c 1-7)

serve:
	@mkdir -p ./bin
	@gowatch -o ./bin/pickaxx -p ./cmd/

clean:
	@rm -rf ./bin/*

test:
	@go test ./...

list:
	@$(MAKE) -rpn | sed -n -e '/^$$/ { n ; /^[^ .#][^ ]*:/ { s/:.*$$// ; p ; } ; }' | egrep --color '^[^ ]*'

.PHONY: serve clean test list

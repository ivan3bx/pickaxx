VERSION     ?= $(shell git log -n 1 --format=%h)
GIT_HASH    := $(shell echo $(VERSION) | cut -c 1-7)
GO_FILES     = $(shell find . -name vendor -prune -o -name '*.go' -print)

serve:
	@mkdir -p ./bin
	@gowatch -o ./bin/pickaxx -p ./cmd/

test:
	@go test ./...

test-convey:
	@goconvey -launchBrowser=false -watchedSuffixes ".json,.go,.yml" -excludedDirs "vendor,bin,testserver,public,templates"

# List all makefile targets
list:
	@$(MAKE) -rpn | sed -n -e '/^$$/ { n ; /^[^ .#][^ ]*:/ { s/:.*$$// ; p ; } ; }' | egrep --color '^[^ ]*'

.PHONY: serve kill restart test clean list

VERSION     ?= $(shell git log -n 1 --format=%h)
GIT_HASH    := $(shell echo $(VERSION) | cut -c 1-7)

serve:
	@mkdir -p ./dist
	@packr2 clean
	@gowatch

clean:
	@packr2 clean
	@rm -rf ./dist/*

test:
ifneq (,$(shell which staticcheck))
	$(shell staticcheck ./...)
else
	@echo "skipping go check"
endif

ifeq (arm,$(shell go env GOARCH))
	go test ./...
else
	go test -race ./...
endif

list:
	@$(MAKE) -rpn | sed -n -e '/^$$/ { n ; /^[^ .#][^ ]*:/ { s/:.*$$// ; p ; } ; }' | egrep --color '^[^ ]*'

.PHONY: serve clean test list

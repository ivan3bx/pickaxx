VERSION     ?= $(shell git log -n 1 --format=%h)
GIT_HASH    := $(shell echo $(VERSION) | cut -c 1-7)

serve: clean
	@mkdir -p ./dist
	@gowatch

clean:
	@packr2 clean
	@rm -rf ./client/dist/*
	@rm -rf ./dist/*

test:
ifneq (,$(shell which staticcheck))
	$(shell staticcheck ./...)
else
	@echo "skipping go check"
endif
	go test -race ./...

list:
	@$(MAKE) -rpn | sed -n -e '/^$$/ { n ; /^[^ .#][^ ]*:/ { s/:.*$$// ; p ; } ; }' | egrep --color '^[^ ]*'

.PHONY: serve clean test list

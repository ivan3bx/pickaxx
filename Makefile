VERSION     ?= $(shell git log -n 1 --format=%h)
GIT_HASH    := $(shell echo $(VERSION) | cut -c 1-7)

serve:
	@mkdir -p ./bin
	@packr2 clean
	@gowatch -o ./bin/pickaxx -p ./cmd/

clean:
	@rm -rf ./bin/*

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

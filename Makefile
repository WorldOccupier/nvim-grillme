QUESTION ?= What should happen when authentication expires?

.PHONY: demo test

demo:
	@GRILLME_PLUGIN_PATH="$(CURDIR)" go run . ask "$(QUESTION)"

test:
	@go test ./...

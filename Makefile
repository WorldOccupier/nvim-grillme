QUESTIONS ?= "What should happen when authentication expires?" "Should retries use backoff?" "What should the user see?"

.PHONY: demo test

demo:
	@GRILLME_PLUGIN_PATH="$(CURDIR)" go run . ask $(QUESTIONS)

test:
	@go test ./...
	@nvim --headless -l tests/grillme.lua

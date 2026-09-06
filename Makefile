QUESTIONS ?= \
	"What should happen when authentication expires?" --recommended "Clear the session and return to the sign-in screen." \
	"Should retries use backoff?" --recommended "Yes. Use capped exponential backoff." \
	"What should the user see?" --recommended "Show a brief error with a retry action."

.PHONY: demo test

demo:
	@GRILLME_PLUGIN_PATH="$(CURDIR)" go run ./cmd/grillme ask $(QUESTIONS)

test:
	@go test ./...
	@nvim --headless -l tests/grillme.lua

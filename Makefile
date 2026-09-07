QUESTIONS ?= \
	"What should happen when authentication expires?" --recommended "Clear the session and return to the sign-in screen." \
	"Should retries use backoff?" --recommended "Yes. Use capped exponential backoff." \
	"What should the user see?" --recommended "Show a brief error with a retry action."

.PHONY: check demo test

demo:
	@GRILLME_PLUGIN_PATH="$(CURDIR)" go run ./cmd/grillme ask $(QUESTIONS)
	@go run ./cmd/grillme clean

test:
	@go test ./...
	@for test in tests/session.lua tests/view.lua tests/grillme.lua; do nvim --headless -l $$test || exit; done

check:
	@go test -race ./...
	@go vet ./...
	@for test in tests/session.lua tests/view.lua tests/grillme.lua; do nvim --headless -l $$test || exit; done
	@git diff --check

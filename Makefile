# # This will output the help for each task
# # thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help: ## This help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help


.PHONY: build
build: ## Build an app's binary.
	@go build -o /dev/null cmd/$(APP_NAME)/server.go

.PHONY: build-image
build-image: ## Build an app's docker image.
	@docker build -t $(APP_NAME) -f deploy/Dockerfile.$(APP_NAME) .

.PHONY: build-all
build-all: ## Build binaries of all apps in cmd/.
	@ls -d cmd/* | xargs -I %d go build -o /dev/null %d/main.go

.PHONY: build-image-nc
build-nc: ## Build an app's docker image without caching.
	@docker build --no-cache -t $(APP_NAME) -f deploy/Dockerfile.$(APP_NAME) .

.PHONY: up
up: ## Launch rustcron/crons in docker-compose.
	@docker-compose -f deploy/docker-compose.yaml up

.PHONY: up-nc
up-nc: ## Launch rustcron/crons in docker-compose without caching.
	@docker-compose -f deploy/docker-compose.yaml up --build

.PHONY: down
down: ## Shutdown rustcrons/crons in docker-compose.
	@docker-compose -f deploy/docker-compose.yaml down

.PHONY: lint
lint: ## Lint repo using golangci-lint. See .golangci.yml for configuration.
	@golangci-lint run


.PHONY: test-server-manager
test-server-manager: ## Integration test server package against AWS.
	@go test -v -count=1 -tags=awsintegration ./cmd/cronman/server

.PHONY: test-mailgun
test-mailgun: ## Integration test email package against mailgun.
	@go test -v -count=1 -tags=mailgunintegration ./internal/email

.PHONY: test-rcon
test-rcon: ## Integration test rcon package against Rust server running in Docker.
	@docker build -t rustpm/rust -f cmd/cronman/rcon/Dockerfile.rust .
	@docker run -dit -p 28016:28016 --rm --name test-rcon-rust rustpm/rust
	@go test -v -count=1 -tags=rconintegration ./cmd/cronman/rcon
	@docker stop test-rcon-rust

.PHONY: test-user
test-user: ## Integration test user API.
	@docker-compose -f cmd/user/rest/docker-compose.yml up --build -V --abort-on-container-exit --exit-code-from test

.PHONY: test-payment
test-payment: ## Integration test payment API.
	@docker-compose -f cmd/payment/rest/docker-compose.yml up --build -V --abort-on-container-exit --exit-code-from test

.PHONY: test-session
test-session: ## Integration test session package.
	@docker-compose -f internal/session/integration_test/docker-compose.yml up --build -V --abort-on-container-exit --exit-code-from test

.PHONY: test-stream
test-stream: ## Integration test stream package.
	@docker-compose -f internal/stream/docker-compose.yml up --build -V --abort-on-container-exit --exit-code-from test

.PHONY: test-staging
test-staging: ## Integration test staging package.
	@docker-compose -f cmd/payment/staging/docker-compose.yml up --build -V --abort-on-container-exit --exit-code-from test

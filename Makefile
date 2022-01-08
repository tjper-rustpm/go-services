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

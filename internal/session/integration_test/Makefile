# # This will output the help for each task
# # thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help: ## This help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

clean: ## Tear down the integration testing environment.
	@docker-compose down --volumes

test: ## Build and run integration test.
	@docker-compose -f docker-compose.yml up --build --renew-anon-volumes --abort-on-container-exit

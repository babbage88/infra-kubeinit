DOCKER_HUB:=ghcr.io/babbage88/kubeinit:
DOCKER_HUB_TEST:=jtrahan88/kubeinit-test:
ENV_FILE:=.env
MIG:=$(shell date '+%m%d%Y.%H%M%S')
SHELL := /bin/bash

check-builder:
	@if ! docker buildx inspect kubeinitbuilder > /dev/null 2>&1; then \
		echo "Builder kubeinitbuilder does not exist. Creating..."; \
		docker buildx create --name kubeinitbuilder --bootstrap; \
	fi

create-builder: check-builder

buildandpush: check-builder
	docker buildx use kubeinitbuilder
	docker buildx build --platform linux/amd64,linux/arm64 -t $(DOCKER_HUB)$(tag) . --push

deploydev: buildandpushdev
	kubectl apply -f deployment/kubernetes/infra-kubeinit.yaml
	kubectl rollout restart deployment infra-kubeinit


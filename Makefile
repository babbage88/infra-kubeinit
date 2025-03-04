DOCKER_HUB:=ghcr.io/babbage88/kubeinit:
DOCKER_HUB_TEST:=jtrahan88/kubeinit-test:
ENV_FILE:=.env
MIG:=$(shell date '+%m%d%Y.%H%M%S')
SHELL := /bin/bash
DB_BUILD_DIR:=~/projects/infra-db
KUBE_INIT_BUILDER:=infra-kube-builder
DB_INIT_BUILDER:=initbuilder-db
DB_BUILDER:=infradb-builder
DB_INIT_IMG:=ghcr.io/babbage88/jobcheck:
DB_IMG:=ghcr.io/babbage88/init-infradb:
curdir:=$(shell pwd)
infradb-deployfile:=deployment/kubernetes/infra-db.yaml
tag:=$(shell git rev-parse HEAD) 
check-builder:
	@if ! docker buildx inspect $(KUBE_INIT_BUILDER) > /dev/null 2>&1; then \
		echo "Builder $(KUBE_INIT_BUILDER) does not exist. Creating..."; \
		docker buildx create --name $(KUBE_INIT_BUILDER) --bootstrap; \
	fi

check-db-builder:
	@if ! docker buildx inspect $(DB_BUILDER) > /dev/null 2>&1; then \
		echo "Builder $(DB_BUILDER) does not exist. Creating..."; \
		docker buildx create --name $(DB_BUILDER) --bootstrap; \
	fi

check-init-builder:
	@if ! docker buildx inspect $(DB_INIT_BUILDER) > /dev/null 2>&1; then \
		echo "Builder $(DB_INIT_BUILDER) does not exist. Creating..."; \
	 	docker buildx create --name $(DB_INIT_BUILDER) --bootstrap; \
	fi

create-builder: check-builder

create-db-builder: check-db-builder

create-init-builder: check-init-builder

buildinitcontainer-db: create-init-builder
	@echo "Building init container image: $(DB_INIT_IMG)$(tag)"
	docker buildx use $(DB_INIT_BUILDER)
	cd $(DB_BUILD_DIR) && docker buildx build --file=Init.Dockerfile --platform linux/amd64,linux/arm64 -t $(DB_INIT_IMG)$(tag) . --push && cd $(curdir)

buildandpush-db: create-db-builder
	@echo "Building image: $(DOCKER_HUB)$(tag)"
	docker buildx use $(DB_BUILDER)
	cd $(DB_BUILD_DIR) && docker buildx build --platform linux/amd64,linux/arm64 -t $(DB_IMG)$(tag) . --push && cd $(curdir)

buildandpush: check-builder
	docker buildx use $(KUBE_INIT_BUILDER)
	docker buildx build --platform linux/amd64,linux/arm64 -t $(DOCKER_HUB)$(tag) . --push

deploydev: buildandpushdev
	kubectl apply -f deployment/kubernetes/infra-kubeinit.yaml
	kubectl rollout restart deployment infra-kubeinit


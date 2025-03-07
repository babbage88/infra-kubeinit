DOCKER_HUB:=ghcr.io/babbage88/kubeinit:
DOCKER_HUB_TEST:=jtrahan88/kubeinit-test:
ENV_FILE:=.env
MIG:=$(shell date '+%m%d%Y.%H%M%S')
SHELL := /bin/bash
DB_BUILD_DIR:=~/projects/infra-db
KUBE_INIT_BUILDER:=infra-kube-builder
DB_BUILDER:=infradb-builder
DB_IMG:=ghcr.io/babbage88/init-infradb:
curdir:=$(shell pwd)
infradb-deployfile:=deployment/kubernetes/infra-db.yaml
tag:=$(shell git rev-parse HEAD) 
MAIN_BRANCH:=master
VERSION_TYPE:=patch
export LATEST_TAG := $(shell git fetch --tags && git tag -l "v[0-9]*.[0-9]*.[0-9]*" | sort -V | tail -n 1)


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

create-builder: check-builder

create-db-builder: check-db-builder

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

# Usage: make release [VERSION=major|minor|patch]
fetch-tags:
	@{ \
	  branch=$$(git rev-parse --abbrev-ref HEAD); \
	  if [ "$$branch" != "$(MAIN_BRANCH)" ]; then \
	    echo "Error: You must be on the $(MAIN_BRANCH) branch. Current branch is '$$branch'."; \
	    exit 1; \
	  fi; \
	  git fetch origin $(MAIN_BRANCH); \
	  UPSTREAM=origin/$(MAIN_BRANCH); \
	  LOCAL=$$(git rev-parse @); \
	  REMOTE=$$(git rev-parse "$$UPSTREAM"); \
	  BASE=$$(git merge-base @ "$$UPSTREAM"); \
	  if [ "$$LOCAL" != "$$REMOTE" ]; then \
	    echo "Error: Your local $(MAIN_BRANCH) branch is not up-to-date with remote. Please pull the latest changes."; \
	    exit 1; \
	  fi; \
	  git fetch --tags; \
	}

release: fetch-tags
	@{ \
	  echo "Latest tag: $(LATEST_TAG)"; \
	  new_tag=$$(go run . --bumper --latest-version "$(LATEST_TAG)" --increment-type=$(VERSION_TYPE)); \
	  echo "Creating new tag: $$new_tag"; \
	  git tag -a $$new_tag -m $$new_tag && git push --tags; \
	}



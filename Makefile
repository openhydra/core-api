REGISTRY ?= registry.cn-shanghai.aliyuncs.com/openhydra
TAG ?=

IMAGETAG ?= $(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --verify HEAD)-$(shell date -u '+%Y%m%d%I%M%S')
BRANCH ?= $(shell git branch --show-current)
ASSERTS ?= $(PWD)/asserts
COMMIT_REF ?= $(shell git rev-parse --verify HEAD)

TAG ?=
ifeq ($(TAG),)
	TAG = $(COMMIT_REF)
endif

.PHONY: test-all
test-all:
	ginkgo -r -v --cover --coverprofile=coverage.out cmd pkg

.PHONY: fmt
fmt:
	gofmt -w pkg cmd

.PHONY: go-build
go-build:
	go build -o cmd/core-api-server/core-api-server -ldflags "-X 'main.version=${TAG}'" cmd/core-api-server/main.go

.PHONY: image
image: go-build
	docker build -f hack/builder/Dockerfile -t $(REGISTRY)/core-api-server:$(IMAGETAG) .

.PHONY: image-then-push
image-then-push: go-build
	docker build -f hack/builder/Dockerfile -t $(REGISTRY)/core-api-server:$(IMAGETAG) . 
	docker tag $(REGISTRY)/core-api-server:$(IMAGETAG) $(REGISTRY)/core-api-server:$(BRANCH)
	docker push $(REGISTRY)/core-api-server:$(IMAGETAG)
	docker push $(REGISTRY)/core-api-server:$(BRANCH)

.PHONY: update-api-doc
update-api-doc:
	swag init --parseDependency -g pkg/north/api/route/handler.go

.PHONY: pre-commit
pre-commit: fmt test-all update-api-doc

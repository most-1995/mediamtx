BASE_IMAGE = golang:1.21-alpine3.19
LINT_IMAGE = golangci/golangci-lint:v1.56.2
NODE_IMAGE = node:20-alpine3.19
ALPINE_IMAGE = alpine:3.19
RPI32_IMAGE = balenalib/raspberry-pi:bullseye-run-20230712
RPI64_IMAGE = balenalib/raspberrypi3-64:bullseye-run-20230530
REGISTRY := 312933510661.dkr.ecr.ap-southeast-1.amazonaws.com
IMAGE_NAME := scm-mediamtx

.PHONY: $(shell ls)

help:
	@echo "usage: make [action]"
	@echo ""
	@echo "available actions:"
	@echo ""
	@echo "  mod-tidy         run go mod tidy"
	@echo "  format           format source files"
	@echo "  test             run tests"
	@echo "  test32           run tests on a 32-bit system"
	@echo "  test-highlevel   run high-level tests"
	@echo "  lint             run linters"
	@echo "  bench NAME=n     run bench environment"
	@echo "  run              run app"
	@echo "  apidocs-lint     run api docs linters"
	@echo "  apidocs-gen      generate api docs HTML"
	@echo "  binaries         build binaries for all platforms"
	@echo "  dockerhub        build and push images to Docker Hub"
	@echo "  dockerhub-legacy build and push images to Docker Hub (legacy)"
	@echo ""

blank :=
define NL

$(blank)
endef

include scripts/*.mk

docker-build:
	docker build --platform linux/amd64 -t $(IMAGE_NAME) .

deploy-ecr:
	aws ecr get-login-password --region ap-southeast-1 --profile scmprofile | docker login --username AWS --password-stdin $(REGISTRY)
	docker tag $(IMAGE_NAME):latest $(REGISTRY)/$(IMAGE_NAME):latest
	docker push $(REGISTRY)/$(IMAGE_NAME):latest

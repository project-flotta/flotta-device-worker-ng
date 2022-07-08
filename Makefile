VERSION = 0.1.0
RELEASE = 1
DIST_DIR = $(shell pwd)/dist
CGO_ENABLED = 0
OS :=$(shell awk -F= '/^ID/{print $$2}' /etc/os-release)
BUILDROOT ?=
NAME ?= device-worker-ng

DOCKER ?= podman
IMG ?= quay.io/ctupangiu/edgedevice-ng:latest
VERSION ?= 0.1
GIT_COMMIT=$(shell git rev-list -1 HEAD --abbrev-commit)

IMAGE_TAG=$(VERSION)-$(GIT_COMMIT)
IMAGE_NAME=device-worker-ng

export GOFLAGS=-mod=vendor -tags=containers_image_openpgp

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


##@ Development

generate.tools:
ifeq (, $(shell which mockery))
	(cd /tmp && go install github.com/vektra/mockery/...@v1.1.2)
endif
ifeq (, $(shell which mockgen))
	(cd /tmp/ && go install github.com/golang/mock/mockgen@v1.6.0)
endif
	@exit

generate: generate-tools
	go generate ./...

GOVER = $(shell pwd)/bin/gover
gover:
ifeq (, $(shell which ginkgo 2> /dev/null))
	$(call go-install-tool,$(GOVER),github.com/sozorogami/gover)
endif

GINKGO = $(shell pwd)/bin/ginkgo
ginkgo:
ifeq (, $(shell which ginkgo 2> /dev/null))
	$(call go-install-tool,$(GINKGO),github.com/onsi/ginkgo/v2/ginkgo@v2.1.3)
else
	GINKGO=$(shell which ginkgo)
endif

test.tools: ## Install test-tools
test.tools: ginkgo gover

gosec: ## Run gosec locally
	$(DOCKER) run --rm -it -v $(PWD):/opt/data/:z docker.io/securego/gosec -exclude-generated /opt/data/...

test: ## Run unit test on device worker
test: test-tools
	$(GINKGO) --race -r $(GINKGO_OPTIONS) ./internal/* ./cmd/*

TEST_IMAGE_NAME ?= device-worker-test
TEST_IMAGE_TAG ?= latest
test.docker:
	$(DOCKER) build tools/ -f Dockerfile_test -t $(TEST_IMAGE_NAME):$(TEST_IMAGE_TAG)
	$(DOCKER) run -v $(PWD):/device-worker --rm $(TEST_IMAGE_NAME):$(TEST_IMAGE_TAG)

test.coverage:
test.coverage: ## Run test and launch coverage tool
test.coverage: GINKGO_OPTIONS ?= --cover
test.coverage: test
	gover
	go tool cover -html gover.coverprofile

test.coverage.clean:
	git ls-files --others --ignored --exclude-standard | grep "coverprofile$$" | xargs rm

.PHONY: vendor

vendor:
	go mod tidy
	go mod vendor

bump.operator: ## Bump flotta operator version dependency to the latest main
	$(eval OPARATOR_VERSION := $(shell curl -s https://api.github.com/repos/project-flotta/flotta-operator/commits/main | jq '.sha' -r))
	go get -d github.com/project-flotta/flotta-operator@$(OPARATOR_VERSION)

clean: ## Clean project
	go mod tidy
	rm -rf bin

##@ Build

build.debug: ## Build with race conditions and lock checker
build.debug: BUILD_OPTIONS=--race
build.debug: CGO_ENABLED=1
build.debug: build

build: ## Build device worker
build:
	mkdir -p ./bin
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_OPTIONS) -o ./bin/device-worker ./main.go

run: build
	./bin/device-worker --config $(PWD)/resources/config.yaml

build.docker:
	$(DOCKER) build . -t $(IMAGE_NAME):$(IMAGE_TAG)

# go-install-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

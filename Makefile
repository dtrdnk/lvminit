APP_NAME := lvminit
IMAGE_NAME ?= lvminit:latest


GOOS ?= linux
GOARCH ?= amd64
GOARM ?=
CGO_ENABLED ?= 0
TAGS := -tags 'osusergo netgo static_build'

GO111MODULE := on
KUBECONFIG := $(HOME)/.kube/config

SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
# gnu date format iso-8601 is parsable with Go RFC3339
BUILDDATE := $(shell date --iso-8601=seconds)
VERSION := $(or ${VERSION},$(shell git describe --tags --exact-match 2> /dev/null || git symbolic-ref -q --short HEAD || git rev-parse --short HEAD))

ifeq ($(CI),true)
  DOCKER_TTY_ARG=
else
  DOCKER_TTY_ARG=t
endif

ifeq ($(CGO_ENABLED),1)
ifeq ($(GOOS),linux)
	LINKMODE := -linkmode external -extldflags '-static -s -w'
endif
endif

.PHONY: all deps build docker-build e2e clean

all: build

deps:
	go mod tidy

build: deps
	GOOS=linux CGO_ENABLED=0 go build -o $(APP_NAME) main.go

docker-build: build
	docker build -t $(IMAGE_NAME) .

/dev/loop%:
	dd if=/dev/zero of=loop$*.img bs=1G seek=4 count=0

ifndef GITHUB_ACTIONS
	@sudo mknod $@ b 7 $*
endif
	@sudo losetup $@ loop$*.img
	@sudo losetup $@

rm-loop%:
	@sudo losetup -d /dev/loop$* || true
	@! losetup /dev/loop$*
	@sudo rm -f /dev/loop$*
	@rm -f loop$*.img
# If removing this loop device fails, you may need to:
# 	sudo dmsetup info
# 	sudo dmsetup remove <DEVICE_NAME>

.PHONY: kind
kind:
	@if ! which kind > /dev/null; then echo "kind needs to be installed"; exit 1; fi
	@if ! kind get clusters | grep lvminit-e2e > /dev/null; then \
		kind create cluster \
		  --name lvminit-e2e \
		  --config tests/kind.yaml; fi

.PHONY: kind-load
kind-load:
	@kind --name lvminit-e2e load docker-image $(IMAGE_NAME)

.PHONY: rm-kind
rm-kind:
	@kind delete cluster --name lvminit-e2e

.PHONY: e2e
e2e: docker-build /dev/loop100 /dev/loop101 kind kind-load
	@cd tests && docker build -t csi-bats . && cd -
	docker run -i$(DOCKER_TTY_ARG) \
		-v "$(KUBECONFIG):/root/.kube/config" \
		-v "$(PWD)/tests:/code" \
		-v "$(PWD)/helm:/helm" \
		--network host \
		csi-bats \
		--verbose-run --trace --timing bats/e2e.bats ; \

.PHONY: test-cleanup
test-cleanup: rm-kind

.PHONY: clean
clean: test-cleanup
	rm -f $(APP_NAME)

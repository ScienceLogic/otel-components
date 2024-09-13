# very simplified version of https://github.com/open-telemetry/opentelemetry-collector-releases/blob/main/Makefile
include ./Makefile.Common
#variables
OTEL_COl_VER?=v0.91.0
DOCKER_TAG?=0.0.0-$(VERSION)
GROUP?=all
PACKAGE_NAME := github.com/sciencelogic/otel-components

ifeq ($(GOOS),windows)
	EXTENSION := .exe
endif

VERSION=$(shell git describe --always --match "v[0-9]*" HEAD)

FIND_MOD_ARGS= -type f -name "go.mod" -not -path "./dist/*" -not -path "./internal/*"
TO_MOD_DIR=dirname {} \; | sort | grep -E '^./'
VERSION=$(shell git describe --always --match "v[0-9]*" HEAD)
FOR_GROUP_TARGET=for-$(GROUP)-target
MOD_NAME=github.com/sciencelogic/otel-components

ALL_MODS := $(shell find ./* $(FIND_MOD_ARGS) -exec $(TO_MOD_DIR) )

.DEFAULT_GOAL := all

all-groups:
	@echo "\nall: $(ALL_MODS)"

.PHONY: all
all: install-tools gotidy gofmt gogovulncheck gotest

.PHONY: gotidy
gotidy:
	$(MAKE) $(FOR_GROUP_TARGET) TARGET="tidy"

.PHONY: gomoddownload
gomoddownload:
	$(MAKE) $(FOR_GROUP_TARGET) TARGET="moddownload"

.PHONY: gotest
gotest:
	$(MAKE) $(FOR_GROUP_TARGET) TARGET="test"

.PHONY: gotest-with-cover
gotest-with-cover:
	@$(MAKE) $(FOR_GROUP_TARGET) TARGET="test-with-cover"

.PHONY: gofmt
gofmt:
	$(MAKE) $(FOR_GROUP_TARGET) TARGET="fmt"

.PHONY: golint
golint:
	$(MAKE) $(FOR_GROUP_TARGET) TARGET="lint"


.PHONY: goporto
goporto: $(PORTO)
	$(PORTO) -w --include-internal --skip-dirs "^cmd$$" ./


# Define a delegation target for each module
.PHONY: $(ALL_MODS)
$(ALL_MODS):
	@echo "Running target '$(TARGET)' in module '$@' as part of group '$(GROUP)'"
	$(MAKE) -C $@ $(TARGET) MOD=$@

.PHONY: for-all-target
for-all-target: $(ALL_MODS)

# Debugging target, which helps to quickly determine whether for-all-target is working or not.
.PHONY: all-pwd
all-pwd:
	$(MAKE) $(FOR_GROUP_TARGET) TARGET="pwd"
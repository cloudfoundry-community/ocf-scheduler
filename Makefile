# This how we want to name the binary output
APP_NAME       ?= scheduler
PROJECT        :=ocf-scheduler
CMD_PATH       ?= cmd
SHELL          :=/bin/bash
GOOS           :=$(shell go env GOOS)
GOARCH         :=$(shell go env GOARCH)
GOMODULECMD    :=main
RELEASE_ROOT   ?=releases
TARGETS        ?=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
MODULE         ?= github.com/cloudfoundry-community/ocf-scheduler

CGO_ENABLED    ?= 0
BUILD_PATH     =builds
BUILD          =${APP_NAME}-${SEMVER_VERSION}
TESTFILES      =`go list ./... | grep -v /vendor/`

define is_not_number 
$(shell echo ${1} | sed -e 's/[0123456789]//g')
endef

ifneq ($(VERSION),)
VERSION_SPLIT:=$(subst ., ,$(VERSION))
  ifneq ($(words $(VERSION_SPLIT)),3)
    $(error VERSION does not have 3 parts |$(words $(VERSION_SPLIT))|$(VERSION)|$(VERSION_SPLIT)|)
  endif
else
VERSION_TAG:=$(shell (git describe --tags --abbrev=0 2>/dev/null || echo 0.0.0) | sed -e "s/^v//")
VERSION_SPLIT:=$(subst ., ,$(VERSION_TAG))
  ifneq ($(words $(VERSION_SPLIT)),3)
    $(error VERSION_TAG does not have 3 parts |$(words $(VERSION_SPLIT))|$(VERSION_TAG)|$(VERSION_SPLIT)|)
  endif

  ifneq ($(words $(call is_not_number,$(word 3,$(VERSION_SPLIT)))), 0)
    $(error The VERSION_TAG patch version string contain non-numeric characters)
  endif

  VERSION_SPLIT:=$(wordlist 1, 2, $(VERSION_SPLIT)) $(shell echo $$(($(word 3,$(VERSION_SPLIT))+1)))
endif

ifneq ($(words $(call is_not_number,$(VERSION_SPLIT))), 0)
  $(error The version string contain non-numeric characters)
endif

SEMVER_MAJOR    ?=$(word 1,$(VERSION_SPLIT))
SEMVER_MINOR    ?=$(word 2,$(VERSION_SPLIT))
SEMVER_PATCH    ?=$(word 3,$(VERSION_SPLIT))
SEMVER_PRERELEASE ?=
SEMVER_BUILDMETA  ?=
BUILD_DATE        :=$(shell date -u -Iseconds)
BUILD_VCS_URL     :=$(shell git config --get remote.origin.url)
BUILD_VCS_ID      :=$(shell git log -n 1 --date=iso-strict-local --format="%h")
BUILD_VCS_ID_DATE :=$(shell TZ=UTC0 git log -n 1 --date=iso-strict-local --format='%ad')

build: SEMVER_PRERELEASE := dev

GO_LDFLAGS = -X '$(GOMODULECMD).SemVerMajor=$(SEMVER_MAJOR)' \
	         -X '$(GOMODULECMD).SemVerMinor=$(SEMVER_MINOR)' \
	         -X '$(GOMODULECMD).SemVerPatch=$(SEMVER_PATCH)' \
	         -X '$(GOMODULECMD).SemVerPrerelease=$(SEMVER_PRERELEASE)' \
	         -X '$(GOMODULECMD).SemVerBuild=$(SEMVER_BUILDMETA)' \
	         -X '$(GOMODULECMD).BuildDate=$(BUILD_DATE)' \
	         -X '$(GOMODULECMD).BuildVcsUrl=$(BUILD_VCS_URL)' \
	         -X '$(GOMODULECMD).BuildVcsId=$(BUILD_VCS_ID)' \
		     -X '$(GOMODULECMD).BuildVcsIdDate=$(BUILD_VCS_ID_DATE)'

# The build meta data is added when the build is done
#
SEMVER_VERSION := $(if $(SEMVER_MAJOR),$(SEMVER_MAJOR),$(error Missing SEMVER_MAJOR))
SEMVER_VERSION := $(SEMVER_VERSION)$(if $(SEMVER_MINOR),.$(SEMVER_MINOR),$(error Missing SEMVER_MINOR))
SEMVER_VERSION := $(SEMVER_VERSION)$(if $(SEMVER_PATCH),.$(SEMVER_PATCH),$(error Missing SEMVER_PATCH))
SEMVER_VERSION := $(SEMVER_VERSION)$(if $(SEMVER_PRERELEASE),-$(SEMVER_PRERELEASE))

MODULE ?= github.com/cloudfoundry-community/ocf-scheduler
CMD_PATH ?= cmd
CGO_ENABLED ?= 0
BUILD_PATH=builds
BUILD=${APP_NAME}-${SEMVER_VERSION}
TARGETS        ?=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
TESTFILES=`go list ./... | grep -v /vendor/`

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS := -w -s -X main.Version=$(VERSION)
ifeq ($(CGO_ENABLED),0)
LDFLAGS += -extldflags '-static'
endif

# Build for the current platform
all: clean build

# Build a new release
release: distclean distbuild linux package

# Builds the project
build:
	go build -ldflags="${GO_LDFLAGS}" -o ./  "./${CMD_PATH}/tzlist/..." "./${CMD_PATH}/scheduler/..."

cli:
	$(MAKE) build APP_NAME=sch CMD_PATH=cmd/cli


# Builds the project for all possible platforms
distbuild:
	mkdir -p ${BUILD_PATH}/linux-amd64-${SEMVER_VERSION}
	mkdir -p ${BUILD_PATH}/linux-arm64-${SEMVER_VERSION}

# Installs our project: copies binaries
install:
	go install -ldflags="${LDFLAGS}"

# Cleans our project: deletes binaries
clean:
	rm -f ./tzlist ./scheduler

# Cleans release files
distclean:
	rm -rf ${BUILD_PATH} ${APP_NAME}-*.tar.gz

test:
	./scripts/blanket

RELEASES := $(foreach target,$(TARGETS),release-$(target)-$(PROJECT))

linux:
	CGO_ENABLED=${CGO_ENABLED} GOOS=linux GOARCH=amd64 go build -ldflags="${GO_LDFLAGS}" -o "${BUILD_PATH}/linux-amd64-${SEMVER_VERSION}" "./${CMD_PATH}/tzlist/..." "./${CMD_PATH}/scheduler/..."
	CGO_ENABLED=${CGO_ENABLED} GOOS=linux GOARCH=arm64 go build -ldflags="${GO_LDFLAGS}" -o "${BUILD_PATH}/linux-arm64-${SEMVER_VERSION}" "./${CMD_PATH}/tzlist/..." "./${CMD_PATH}/scheduler/..."

package:
	# cd ${BUILD_PATH} && tar -z -c -v -f ../${BUILD}.tar.gz "${BUILD_PATH}/*-${SEMVER_VERSION}/*"
	#tar -z -c -v -f ${BUILD}.tar.gz "${BUILD_PATH}/*-${SEMVER_VERSION}/*"
	tar -z -c -v -f ${BUILD}.tar.gz "${BUILD_PATH}/"

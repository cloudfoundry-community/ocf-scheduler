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
TESTFILES      =`go list ./... | grep -v /vendor/`

define is_not_number 
$(shell echo ${1} | sed -e 's/[0123456789]//g')
endef

# Accept both 'vX.Y.Z' (legacy) and 'v-X.Y.Z' (current). Go refuses to
# resolve a 'vN.Y.Z' tag for N>=2 unless the module path carries a '/vN'
# suffix; 'v-N.Y.Z' is not valid semver, so Go never version-parses it and
# it stays usable as a `go get` revision. Strip 'v-' before 'v' — reversed,
# the 'v' rule leaves a leading '-' that HAS_PRERELEASE below then reads as
# a prerelease separator, marking a GA release as a prerelease.
CLEAN_VERSION = $(patsubst v%,%,$(patsubst v-%,%,$(VERSION)))
HAS_BUILDMETA := $(findstring +,$(CLEAN_VERSION))
VERSION_BUILDMETA := $(if $(HAS_BUILDMETA),$(lastword $(subst +, ,$(CLEAN_VERSION))),)

VERSION_AND_PRERELEASE := $(firstword $(subst +, ,$(CLEAN_VERSION)))

HAS_PRERELEASE := $(findstring -,$(VERSION_AND_PRERELEASE))
VERSION_ONLY := $(firstword $(subst -, ,$(VERSION_AND_PRERELEASE)))
VERSION_PRERELEASE := $(if $(HAS_PRERELEASE),$(patsubst $(VERSION_ONLY)-%,%,$(VERSION_AND_PRERELEASE)),)

# Output results

ifneq ($(VERSION_ONLY),)
VERSION_SPLIT:=$(subst ., ,$(VERSION_ONLY))
  ifneq ($(words $(VERSION_SPLIT)),3)
    $(error VERSION does not have 3 parts |$(words $(VERSION_SPLIT))|$(VERSION_ONLY)|$(VERSION_SPLIT)|)
  endif
else
VERSION_TAG:=$(shell git describe --tags --abbrev=0 2>/dev/null || echo 0.0.0)
CLEAN_VERSION_TAG = $(patsubst v%,%,$(patsubst v-%,%,$(VERSION_TAG)))
VERSION_SPLIT:=$(subst ., ,$(CLEAN_VERSION_TAG))
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
SEMVER_PRERELEASE ?=$(VERSION_PRERELEASE)
SEMVER_BUILDMETA  ?=$(VERSION_BUILDMETA)
BUILD_DATE        :=$(shell date -u -Iseconds)
BUILD_VCS_URL     :=$(shell git config --get remote.origin.url)
BUILD_VCS_ID      :=$(shell git log -n 1 --date=iso-strict-local --format="%h")
BUILD_VCS_ID_DATE :=$(shell TZ=UTC0 git log -n 1 --date=iso-strict-local --format='%ad')

build: SEMVER_PRERELEASE := $(or $(SEMVER_PRERELEASE),dev)

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
CGO_ENABLED ?= 0
TARGETS        ?=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
TESTFILES=`go list ./... | grep -v /vendor/`

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS := -w -s -X main.Version=$(VERSION)
ifeq ($(CGO_ENABLED),0)
LDFLAGS += -extldflags '-static'
endif

# Build for the current platform
all: clean build

debug_version:
	@echo VERSION $(VERSION)
	@echo CLEAN_VERSION $(CLEAN_VERSION)
	@echo VERSION_BUILDMETA $(VERSION_BUILDMETA)
	@echo VERSION_AND_PRERELEASE $(VERSION_AND_PRERELEASE)
	@echo VERSION_PRERELEASE $(VERSION_PRERELEASE)
	@echo VERSION_ONLY  $(VERSION_ONLY)
	@echo VERSION_SPLIT  $(VERSION_SPLIT)
	@echo VERSION_TAG  $(VERSION_TAG)
	@echo CLEAN_VERSION_TAG  $(CLEAN_VERSION_TAG)

RELEASES := $(foreach target,$(TARGETS),release-$(target)-$(PROJECT))
PACKAGES := $(foreach target,$(TARGETS),package-$(target)-$(PROJECT))

# Build a new release
release: distclean distbuild $(RELEASES)  $(PACKAGES)

# Builds the project

build:
	go build -ldflags="${GO_LDFLAGS}" -o ./  "./cmd/tzlist/..." "./cmd/scheduler/..."

cli:
	$(MAKE) build APP_NAME=sch CMD_PATH=cmd/cli


# Builds the project for all possible platforms

define distbuild
	@mkdir -p ${RELEASE_ROOT}/${1}-${2}-${SEMVER_VERSION}

endef
distbuild:
	@echo "Building release directories..."
	$(foreach target,$(TARGETS), $(call distbuild,$(word 1, $(subst /, ,$(target))),$(word 2, $(subst /, ,$(target)))))

# Cleans our project: deletes binaries
clean:
	@echo "Cleaning built executables..."
	@rm -f ./tzlist ./scheduler

# Cleans release files
distclean: clean
	@echo "Cleaning ${RELEASE_ROOT} directories..."
	@ rm -rf ${RELEASE_ROOT} ${APP_NAME}-*.tar.gz

test:
	./scripts/blanket

define build-target
release-$(1)/$(2)-$(PROJECT): RELEASE_BUILD_DIR:=$(1)-$(2)-$(SEMVER_VERSION)

release-$(1)/$(2)-$(PROJECT): RELEASE_EXECUTABLE_DIR:=$(RELEASE_ROOT)/$$(RELEASE_BUILD_DIR)

release-$(1)/$(2)-$(PROJECT): RELEASE_GO_LDFLAGS:=-ldflags="$$(GO_LDFLAGS)"

release-$(1)/$(2)-$(PROJECT):
	@echo "Building $$(PROJECT) executables version $$(SEMVER_VERSION) for $(1) $(2) ..."
	@CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go build -o $$(RELEASE_EXECUTABLE_DIR) $$(RELEASE_GO_LDFLAGS) "./cmd/tzlist/..." "./cmd/scheduler/..."
	@echo "Generate $$(PROJECT) digests ..."
	@scripts/shait "$$(RELEASE_EXECUTABLE_DIR)" sha1 sha256

endef

$(foreach target,$(TARGETS), $(eval $(call build-target,$(word 1, $(subst /, ,$(target))),$(word 2, $(subst /, ,$(target))),$(SEMVER_BUILDMETA))))

define package-target
package-$(1)/$(2)-$(PROJECT): RELEASE_BUILD_DIR:=$(1)-$(2)-$(SEMVER_VERSION)

package-$(1)/$(2)-$(PROJECT):
	@echo "Packaging $$(PROJECT) for $$(RELEASE_BUILD_DIR) ..."
	@(cd $(RELEASE_ROOT); tar -z -c -f "${APP_NAME}-$$(RELEASE_BUILD_DIR).tar.gz" "$$(RELEASE_BUILD_DIR)")

endef

$(foreach target,$(TARGETS), $(eval $(call package-target,$(word 1, $(subst /, ,$(target))),$(word 2, $(subst /, ,$(target))))))

# Build all platforms with -work flag, capture WORK directories for later re-linking
release-workdir: distclean
	@WORKDIR_BASE=$$(mktemp -d) && \
	for target in $(TARGETS); do \
		os=$${target%%/*} && \
		arch=$${target##*/} && \
		outdir=$(RELEASE_ROOT)/$$os-$$arch-$(SEMVER_VERSION) && \
		mkdir -p $$outdir && \
		for cmd in scheduler tzlist; do \
			echo "Building $$cmd for $$os/$$arch with -work..." && \
			build_out=$$(CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
				go build -work -ldflags="$(GO_LDFLAGS)" \
				-o $$outdir/ ./cmd/$$cmd/... 2>&1) && \
			workpath=$$(echo "$$build_out" | grep "^WORK=" | cut -d= -f2) && \
			test -n "$$workpath" || { echo "Failed to capture WORK directory"; exit 1; } && \
			echo "  WORK=$$workpath" && \
			platdir=$$WORKDIR_BASE/$$os-$$arch/$$cmd && \
			mkdir -p $$platdir/archives && \
			icl=$$(find $$workpath -name importcfg.link | head -1) && \
			test -n "$$icl" || { echo "No importcfg.link found in $$workpath"; exit 1; } && \
			n=0 && > $$platdir/importcfg.link && \
			while IFS= read -r line; do \
				case "$$line" in \
				"packagefile "*) \
					pkg="$${line%%=*}" && \
					apath="$${line#*=}" && \
					n=$$((n + 1)) && \
					cp "$$apath" "$$platdir/archives/$$n.a" && \
					echo "$$pkg=archives/$$n.a" >> $$platdir/importcfg.link ;; \
				*) \
					echo "$$line" >> $$platdir/importcfg.link ;; \
				esac; \
			done < "$$icl" && \
			grep "cmd/$$cmd=" $$platdir/importcfg.link | head -1 | cut -d= -f2 > $$platdir/main_archive && \
			rm -rf "$$workpath"; \
		done && \
		scripts/shait "$$outdir" sha1 sha256 && \
		(cd $(RELEASE_ROOT) && tar -czf "$(APP_NAME)-$$os-$$arch-$(SEMVER_VERSION).tar.gz" "$$os-$$arch-$(SEMVER_VERSION)"); \
	done && \
	echo "Archiving workdir..." && \
	tar -czf $(RELEASE_ROOT)/$(APP_NAME)-workdir-$(SEMVER_VERSION).tar.gz -C $$WORKDIR_BASE . && \
	rm -rf $$WORKDIR_BASE

# Extract workdir archive and re-link with new ldflags using go tool link directly
relink:
	@test -n "$(WORKDIR_ARCHIVE)" || { echo "WORKDIR_ARCHIVE must be set"; exit 1; }
	@WORKDIR_BASE=$$(mktemp -d) && \
	echo "Extracting workdir archive from $(WORKDIR_ARCHIVE)..." && \
	tar -xzf $(WORKDIR_ARCHIVE) -C $$WORKDIR_BASE && \
	GOTOOLCHAIN=$$(grep -a -m1 -o 'go object [a-z0-9]* [a-z0-9]* go1[0-9.]*' \
		$$(find $$WORKDIR_BASE -name '*.a' | head -1) | awk '{print $$5}') && \
	{ test -n "$$GOTOOLCHAIN" || { echo "Cannot determine Go toolchain from workdir archives" >&2; exit 1; }; } && \
	export GOTOOLCHAIN && \
	echo "Re-linking with toolchain $$GOTOOLCHAIN" && \
	rm -rf $(RELEASE_ROOT) && mkdir -p $(RELEASE_ROOT) && \
	for target in $(TARGETS); do \
		os=$${target%%/*} && \
		arch=$${target##*/} && \
		outdir=$(CURDIR)/$(RELEASE_ROOT)/$$os-$$arch-$(SEMVER_VERSION) && \
		mkdir -p $$outdir && \
		for cmd in scheduler tzlist; do \
			platdir=$$WORKDIR_BASE/$$os-$$arch/$$cmd && \
			main_a=$$(cat $$platdir/main_archive) && \
			echo "Re-linking $$cmd for $$os/$$arch..." && \
			(cd $$platdir && GOOS=$$os GOARCH=$$arch go tool link \
				-importcfg importcfg.link \
				$(GO_LDFLAGS) \
				-buildmode=exe \
				-o $$outdir/$$cmd \
				$$main_a); \
		done && \
		scripts/shait "$$outdir" sha1 sha256 && \
		(cd $(RELEASE_ROOT) && tar -czf "$(APP_NAME)-$$os-$$arch-$(SEMVER_VERSION).tar.gz" "$$os-$$arch-$(SEMVER_VERSION)"); \
	done && \
	rm -rf $$WORKDIR_BASE

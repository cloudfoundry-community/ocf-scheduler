# This how we want to name the binary output
APP_NAME ?= scheduler

# These are the values we want to pass for VERSION and BUILD
# git tag 1.0.1
# git commit -am "One more change after the tags"
VERSION ?= `./scripts/genver` (dev)

MODULE ?= github.com/cloudfoundry-community/ocf-scheduler
CMD_PATH ?= cmd/scheduler
CGO_ENABLED ?= 0
BUILD_PATH=builds
BUILD=${APP_NAME}-${VERSION}
TESTFILES=`go list ./... | grep -v /vendor/`

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS=-w -s -X main.Version=$(VERSION)
ifeq ($(CGO_ENABLED),0)
LDFLAGS=${LDFLAGS} -extldflags '-static'
endif

# Build for the current platform
all: clean build

# Build a new release
release: distclean distbuild linux package

# Builds the project
build:
	go build -ldflags="${LDFLAGS}" -o "${APP_NAME}" "${MODULE}/${CMD_PATH}"

cli:
	$(MAKE) build APP_NAME=sch CMD_PATH=cmd/cli


# Builds the project for all possible platforms
distbuild:
	mkdir -p ${BUILD_PATH}

# Installs our project: copies binaries
install:
	go install -ldflags="${LDFLAGS}"

# Cleans our project: deletes binaries
clean:
	rm -rf ${APP_NAME}

# Cleans release files
distclean:
	rm -rf ${TARGET} ${TARGET}.tar.gz

test:
	./scripts/blanket

linux:
	CGO_ENABLED=${CGO_ENABLED} GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o "${BUILD_PATH}/${BUILD}-linux-amd64" "${MODULE}/${CMD_PATH}"
	CGO_ENABLED=${CGO_ENABLED} GOOS=linux GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o "${BUILD_PATH}/${BUILD}-linux-arm64" "${MODULE}/${CMD_PATH}"

package:
	tar -C builds -z -c -v -f ${TARGET}.tar.gz "${APP_NAME}-${VERSION}"

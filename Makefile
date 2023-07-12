# This how we want to name the binary output
BINARY ?= scheduler

# These are the values we want to pass for VERSION and BUILD
# git tag 1.0.1
# git commit -am "One more change after the tags"
VERSION ?= `./scripts/genver`" (dev)"

MODULE ?= "github.com/cloudfoundry-community/ocf-scheduler"
CMD_PATH ?= "cmd/scheduler"
BUILD_PATH="builds"
BUILD=${BINARY}-${VERSION}"
TESTFILES=`go list ./... | grep -v /vendor/`

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS=-ldflags "-w -s \
				-extldflags '-static'"

# Build for the current platform
all: clean build

# Build a new release
release: distclean distbuild linux package

# Builds the project
build:
	go build ${LDFLAGS} -o ${BINARY} ${MODULE}/$CMD_PATH

cli:
	go build ${LDFLAGS} -o sch ${MODULE}/cmd/cli


# Builds the project for all possible platforms
distbuild:
	mkdir -p ${BUILD_PATH}

# Installs our project: copies binaries
install:
	go install ${LDFLAGS}

# Cleans our project: deletes binaries
clean:
	rm -rf ${BINARY}

# Cleans release files
distclean:
	rm -rf ${TARGET} ${TARGET}.tar.gz

test:
	./scripts/blanket

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_PATH}/${BINARY}-${VERSION}-linux-amd64 ${MODULE}/${CMD_PATH}
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_PATH}/${BINARY}-${VERSION}-linux-arm64 ${MODULE}/${CMD_PATH}

package:
	tar -C builds -z -c -v -f ${TARGET}.tar.gz ${BINARY}-${VERSION}

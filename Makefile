# This how we want to name the binary output
BINARY=scheduler

# These are the values we want to pass for VERSION and BUILD
# git tag 1.0.1
# git commit -am "One more change after the tags"
VERSION=`./scripts/genver`
BUILD=`date +%FT%T%z`
PACKAGE="github.com/starkandwayne/scheduler-for-ocf/cmd/scheduler"
TARGET="builds/${BINARY}-${VERSION}"
PREFIX="${TARGET}/${BINARY}-${VERSION}"
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
	go build ${LDFLAGS} -o ${BINARY} ${PACKAGE}

cli:
	go build ${LDFLAGS} -o sch github.com/starkandwayne/scheduler-for-ocf/cmd/cli


# Builds the project for all possible platforms
distbuild:
	mkdir -p ${TARGET}

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
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${TARGET}/${BINARY}-linux-amd64 ${PACKAGE}
	CGO_ENABLED=0 GOOS=linux GOARCH=arm go build ${LDFLAGS} -o ${TARGET}/${BINARY}-linux-arm ${PACKAGE}

package:
	tar -C builds -z -c -v -f ${TARGET}.tar.gz ${BINARY}-${VERSION}

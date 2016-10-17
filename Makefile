# This is how we want to name the binary output
BASE_BINARY=rltop

# These are the values we want to pass for Version and BuildTime
VERSION := $(shell sh -c 'cat VERSION')
BUILD_TIME=`date +%Y-%m-%d`
GIT_HASH=`git rev-parse --verify HEAD`
UNAME := $(shell uname)
GIT_TAG := $(shell git describe --abbrev=0 --tags)

ifeq ($(UNAME), Linux)
	OS=linux
endif
ifeq ($(UNAME), Darwin)
	OS=darwin
endif
ARCH=amd64

ifeq ($(ADD_VERSION_OS_ARCH), 1)
	BINARY=$(BASE_BINARY)-$(VERSION)-$(OS)-$(ARCH)
else
  BINARY=$(BASE_BINARY)
endif

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS="-s -w -X main.buildDate=`date +%Y-%m-%d` -X main.version=${VERSION} -X main.commitHash=${GIT_HASH}"

build:
	GOOS=${OS} GOARCH=${ARCH} go build -ldflags ${LDFLAGS} -o bin/${BINARY} main.go

# Build the binaries (compressed) for both MacOS and Linux
archives:
	GOOS=darwin GOARCH=amd64 go build -ldflags ${LDFLAGS} -o bin/${BINARY}-${GIT_TAG}-darwin-amd64 main.go
	tar -zcvf bin/${BINARY}-${GIT_TAG}-darwin-amd64.tar.gz bin/${BINARY}-${GIT_TAG}-darwin-amd64
	rm bin/${BINARY}-${GIT_TAG}-darwin-amd64
	GOOS=linux GOARCH=amd64 go build -ldflags ${LDFLAGS} -o bin/${BINARY}-${GIT_TAG}-linux-amd64 main.go
	tar -zcvf bin/${BINARY}-${GIT_TAG}-linux-amd64.tar.gz bin/${BINARY}-${GIT_TAG}-linux-amd64
	rm bin/${BINARY}-${GIT_TAG}-linux-amd64
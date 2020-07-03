BUILDINFO = $(subst ${GOPATH}/src/,,${PWD})

VERSION  = $(shell git describe --always --tags --dirty=-dirty)
REVISION = $(shell git rev-parse HEAD)
BRANCH   = $(shell git rev-parse --abbrev-ref HEAD)

all: build

build:
	@echo ">> building common_exporter"
	@CGO_ENABLED=0 go build -ldflags "\
            -X ${BUILDINFO}.Version=${VERSION} \
            -X ${BUILDINFO}.Revision=${REVISION} \
            -X ${BUILDINFO}.Branch=${BRANCH} \
            -X ${BUILDINFO}.BuildUser=$(USER)@$(HOSTNAME) \
            -X ${BUILDINFO}.BuildDate=$(shell date +%Y-%m-%dT%T%z)"


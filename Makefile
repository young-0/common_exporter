BUILDINFO = $(subst ${GOPATH}/src/,,${PWD})

VERSION  = $(shell git describe --always --tags --dirty=-dirty)
REVISION = $(shell git rev-parse HEAD)
BRANCH   = $(shell git rev-parse --abbrev-ref HEAD)

branch := $(shell git rev-parse --abbrev-ref HEAD)
version := $(shell git describe --tags --always --dirty)
revision := $(shell git rev-parse HEAD)
release := $(shell git describe --tags --always --dirty | cut -d"-" -f 1,2)

GO_LDFLAGS := -X main.Branch=${branch} -X main.Version=${version} -X main.Revision=${revision}
GO_LDFLAGS += -w -s -extldflags "-static"

all: build

build:
	@echo ">> building common_exporter"
	@CGO_ENABLED=0 go build -ldflags "$(GO_LDFLAGS)"


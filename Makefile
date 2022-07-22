NAMESPACE  := logicmonitor
REPOSITORY := collector
VERSION       ?= $(shell git describe --tags --always --dirty)

default: build

gofmt:
ifeq ($(shell uname -s), Darwin)
	find pkg/ -type f | grep go | egrep -v "mocks|gomock" | xargs gofmt -l -d -s -w; sync
	find pkg/ -type f | grep go | egrep -v "mocks|gomock" | xargs gofumpt -l -d -w; sync
	find pkg/ -type f | grep go | egrep -v "mocks|gomock" | xargs gci write; sync
	find pkg/ -type f | grep go | egrep -v "mocks|gomock" | xargs goimports -l -d -w; sync
	find cmd/ -type f | grep go | egrep -v "mocks|gomock" | xargs gofmt -l -d -s -w; sync
	find cmd/ -type f | grep go | egrep -v "mocks|gomock" | xargs gofumpt -l -d -w; sync
	find cmd/ -type f | grep go | egrep -v "mocks|gomock" | xargs gci write; sync
	find cmd/ -type f | grep go | egrep -v "mocks|gomock" | xargs goimports -l -d -w; sync
	gofmt -l -d -s -w main.go; sync
	gofumpt -l -d -w main.go; sync
	gci write main.go; sync
	goimports -l -d -w main.go; sync
endif

build:
	docker build -t $(NAMESPACE)/$(REPOSITORY):$(VERSION) .

gononroot-build:
	docker build -t $(NAMESPACE)/$(REPOSITORY):$(VERSION) -f Dockerfile.gononroot .

nonroot-build:
	docker build -t $(NAMESPACE)/$(REPOSITORY):$(VERSION) -f Dockerfile.nonroot .
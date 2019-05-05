# Env
DEPCMD=dep
DEP_ENSURE=$(DEPCMD) ensure
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
BINARY_NAME=kubectl-glance
export GO111MODULE=on

all: test build

build:
	$(DEP_ENSURE) -v
	$(GOBUILD) -o $(BINARY_NAME) -mod=readonly -v

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)

install: build
	mv ./$(BINARY_NAME) $(GOPATH)/bin/

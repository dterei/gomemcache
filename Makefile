ifndef GO
GO=go
endif

all: memcached

memcached: src/memcached/*.go
	@mkdir -p bin
	@cd bin && GOPATH=$(CURDIR) $(GO) build memcached

memcached-race: src/memcached/*.go
	@mkdir -p bin
	@cd bin && GOPATH=$(CURDIR) $(GO) build -race -o memcached-race memcached

.PHONY: clean
clean:
	@GOPATH=$(CURDIR) $(GO) clean
	@rm -Rf bin bin-debug
	@rm -Rf dist_root

.PHONY: testfull
testfull: test testclient

.PHONY: test
test: memcached
	@GOPATH=$(CURDIR) $(GO) test -race -v memcached

.PHONY: testclient
testclient: memcached-race
	@./bin/memcached-race & echo $$! > test.pids
	@GOPATH=$(CURDIR) GO15VENDOREXPERIMENT=1 $(GO) test -v mc; \
		cd $(CURDIR); cat test.pids | xargs kill; rm test.pids

fmt:
	@GOPATH=$(CURDIR) $(GO) fmt memcached

vet:
	@GOPATH=$(CURDIR) $(GO) vet memcached

lint:
	@GOPATH=$(CURDIR) golint memcached

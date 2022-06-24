VERSION ?= $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)

BINARY ?= cadence-analyzer

.PHONY: test
test:
	go test -v -parallel 4 ./...

.PHONY: binary
binary: $(BINARY)

$(BINARY):
	go build -v -trimpath -o $(BINARY) .

.PHONY: versioned-binaries
versioned-binaries:
	$(MAKE) OS=linux ARCH=amd64 ARCHNAME=x86_64 versioned-binary
	$(MAKE) OS=linux ARCH=arm64 versioned-binary
	$(MAKE) OS=darwin ARCH=amd64 ARCHNAME=x86_64 versioned-binary
	$(MAKE) OS=darwin ARCH=arm64 versioned-binary
	$(MAKE) OS=windows ARCH=amd64 ARCHNAME=x86_64 versioned-binary

.PHONY: versioned-binary
versioned-binary:
	GOOS=$(OS) GOARCH=$(ARCH) $(MAKE) BINARY=cadence-analyzer-$(or ${ARCHNAME},${ARCHNAME},${ARCH})-$(OS)-$(VERSION) binary

.PHONY: publish
publish:
	gsutil -m cp cadence-analyzer-*-$(VERSION) gs://flow-cli

.PHONY: clean
clean:
	rm -f cadence-analyzer*


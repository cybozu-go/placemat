SUDO = sudo
FAKEROOT = fakeroot

TESTTOOLDIR := $(CURDIR)/bin
STATICCHECK := $(TESTTOOLDIR)/staticcheck
CUSTOMCHECKER := $(TESTTOOLDIR)/custom-checker

### for debian package
PACKAGES := fakeroot
WORKDIR := $(CURDIR)/work
CONTROL := $(WORKDIR)/DEBIAN/control
DOCDIR := $(WORKDIR)/usr/share/doc/placemat2
EXAMPLEDIR := $(WORKDIR)/usr/share/doc/placemat2/examples
BASH_COMPLETION_DIR := $(WORKDIR)/etc/bash_completion.d
BINDIR := $(WORKDIR)/usr/bin
VERSION = v0.1
DEB_VERSION = $(patsubst v%,%,$(VERSION))
DEB = placemat2_$(DEB_VERSION)_amd64.deb
DEST = .
SBIN_PKGS = ./cmd/placemat2 ./cmd/pmctl2

.PHONY: deb
deb: $(DEB)

$(DEB):
	rm -rf $(WORKDIR)
	cp -r debian $(WORKDIR)
	sed 's/@VERSION@/$(DEB_VERSION)/' debian/DEBIAN/control > $(CONTROL)
	mkdir -p $(BINDIR)
	GOBIN=$(BINDIR) go install $(SBIN_PKGS)
	mkdir -p $(DOCDIR)
	cp ../README.md ../LICENSE ../docs/pmctl.md $(DOCDIR)
	cp -r ../examples $(DOCDIR)
	mkdir -p $(BASH_COMPLETION_DIR)
	$(BINDIR)/pmctl2 completion bash > $(BASH_COMPLETION_DIR)/placemat2
	chmod -R g-w $(WORKDIR)
	$(FAKEROOT) dpkg-deb --build $(WORKDIR) $(DEST)
	rm -rf $(WORKDIR)

.PHONY: clean
clean:
	rm -rf $(TESTTOOLDIR) $(WORKDIR) $(DEB)

.PHONY: setup
setup:
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends qemu qemu-kvm cloud-utils freeipmi-tools

.PHONY: test-tools
test-tools:
	GOBIN=$(TESTTOOLDIR) go install github.com/cybozu-go/golang-custom-analyzer/cmd/custom-checker@latest
	GOBIN=$(TESTTOOLDIR) go install honnef.co/go/tools/cmd/staticcheck@latest

.PHONY: check-generate
check-generate:
	go mod tidy
	git diff --exit-code --name-only

.PHONY: code-check
code-check:
	test -z "$$(gofmt -s -l . | tee /dev/stderr)"
	$(STATICCHECK) ./...
	test -z "$$($(CUSTOMCHECKER) -restrictpkg.packages=html/template,log $$(go list -tags='$(GOTAGS)' ./...) 2>&1 | tee /dev/stderr)"
	go vet ./...

.PHONY: test
test:
	go test -race -v -count 1 ./...

.PHONY: test-dcnet
test-dcnet:
	go test -c ./pkg/dcnet
	$(SUDO) ./dcnet.test -test.v
	rm -f dcnet.test

.PHONY: test-vm
test-vm:
	go test -c ./pkg/vm
	$(SUDO) ./vm.test -test.v
	rm -f vm.test

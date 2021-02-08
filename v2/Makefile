SUDO = sudo
FAKEROOT = fakeroot

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
	rm -rf $(WORKDIR) $(DEB)

.PHONY: setup
setup:
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends qemu qemu-kvm cloud-utils freeipmi-tools

.PHONY: code-check
code-check: test-tools
	test -z "$$(gofmt -s -l . | tee /dev/stderr)"
	staticcheck ./...
	test -z "$$(nilerr ./... 2>&1 | tee /dev/stderr)"
	test -z "$$(custom-checker -restrictpkg.packages=html/template,log $$(go list -tags='$(GOTAGS)' ./...) 2>&1 | tee /dev/stderr)"
	go install ./...
	go vet ./...

.PHONY: test
test: code-check
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

.PHONY: test-tools
test-tools: custom-checker staticcheck nilerr

.PHONY: custom-checker
custom-checker:
	if ! which custom-checker >/dev/null; then \
		cd /tmp; env GOFLAGS= GO111MODULE=on go get github.com/cybozu/neco-containers/golang/analyzer/cmd/custom-checker; \
	fi

.PHONY: staticcheck
staticcheck:
	if ! which staticcheck >/dev/null; then \
		cd /tmp; env GOFLAGS= GO111MODULE=on go get honnef.co/go/tools/cmd/staticcheck; \
	fi

.PHONY: nilerr
nilerr:
	if ! which nilerr >/dev/null; then \
		cd /tmp; env GOFLAGS= GO111MODULE=on go get github.com/gostaticanalysis/nilerr/cmd/nilerr; \
	fi

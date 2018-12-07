# Makefile for placemat

SUDO = sudo
FAKEROOT = fakeroot
TAGS =

### for Go
GOFLAGS = -mod=vendor
export GOFLAGS

### for debian package
PACKAGES := fakeroot
WORKDIR := $(CURDIR)/work
CONTROL := $(WORKDIR)/DEBIAN/control
DOCDIR := $(WORKDIR)/usr/share/doc/placemat
EXAMPLEDIR := $(WORKDIR)/usr/share/doc/placemat/examples
SBINDIR := $(WORKDIR)/usr/sbin
VERSION = 1.1.0-master
DEB = placemat_$(VERSION)_amd64.deb
DEST = .
SBIN_PKGS = ./pkg/placemat ./pkg/pmctl

test:
	test -z "$$(gofmt -s -l . | grep -v '^vendor' | tee /dev/stderr)"
	golint -set_exit_status $$(go list -tags='$(TAGS)' ./... | grep -v /vendor/)
	go build -tags='$(TAGS)' ./...
	go test -tags='$(TAGS)' -race -v ./...
	go vet -tags='$(TAGS)' ./...

mod:
	go mod tidy
	go mod vendor
	git add -f vendor
	git add go.mod go.sum

deb: $(DEB)

$(DEB):
	rm -rf $(WORKDIR)
	cp -r debian $(WORKDIR)
	sed 's/@VERSION@/$(patsubst v%,%,$(VERSION))/' debian/DEBIAN/control > $(CONTROL)
	mkdir -p $(SBINDIR)
	GOBIN=$(SBINDIR) go install -tags='$(TAGS)' $(SBIN_PKGS)
	mkdir -p $(DOCDIR)
	cp README.md LICENSE docs/pmctl.md $(DOCDIR)
	cp -r examples $(DOCDIR)
	chmod -R g-w $(WORKDIR)
	$(FAKEROOT) dpkg-deb --build $(WORKDIR) $(DEST)
	rm -rf $(WORKDIR)

setup:
	GO111MODULE=off go get -u golang.org/x/lint/golint
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)

clean:
	rm -rf $(WORKDIR) $(DEB)

.PHONY:	all test mod deb setup clean

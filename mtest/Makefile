# Makefile for multi-host testing using ginkgo

# configuration variables
BRIDGE_ADDRESS = 10.0.0.1
NODE1 = 10.0.0.11
NODE2 = 10.0.0.12
GINKGO = $(GOPATH)/bin/ginkgo
PLACEMAT_DATADIR = /var/scratch/placemat

export BRIDGE_ADDRESS NODE1 NODE2 GINKGO

# non-configuration variables
SSH_PRIVKEY := ./mtest_key
OUTPUT := ./output
UBUNTU_IMAGE := ubuntu-18.04-server-cloudimg-amd64.img
PLACEMAT = $(OUTPUT)/placemat
CLUSTER_YAML = $(OUTPUT)/cluster.yml

export SSH_PRIVKEY PLACEMAT CLUSTER_YAML

GENERATED_FILES = $(OUTPUT)/placemat \
	$(OUTPUT)/cluster.yml \
	$(OUTPUT)/user-data_node1.yml \
	$(OUTPUT)/user-data_node2.yml \
	$(OUTPUT)/network1.yml \
	$(OUTPUT)/network2.yml \
	$(UBUNTU_IMAGE)

all: test

$(UBUNTU_IMAGE):
	curl -sSLf -o $@ https://cloud-images.ubuntu.com/releases/18.04/release/$@

$(OUTPUT)/cluster.yml: cluster.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@UBUNTU_IMAGE@/$(UBUNTU_IMAGE)/' $< > $@

$(OUTPUT)/user-data_node1.yml: user-data.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@HOST_NAME@/node1/' $< > $@

$(OUTPUT)/user-data_node2.yml: user-data.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@HOST_NAME@/node2/' $< > $@

$(OUTPUT)/network1.yml: network.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@HOST@/$(NODE1)/' $< > $@

$(OUTPUT)/network2.yml: network.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@HOST@/$(NODE2)/' $< > $@

$(OUTPUT)/placemat:
	mkdir -p $(OUTPUT)
	cd ..; go get ./... && GOBIN=$(abspath $(OUTPUT)) go install ./cmd/$(notdir $@)

test: $(GENERATED_FILES)
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node1
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node2
	$(GINKGO) -v

clean:
	rm -rf $(OUTPUT)

setup:
	go get -t -d ./...
	go get -u github.com/onsi/ginkgo/ginkgo
	go get -u github.com/onsi/gomega/...

.PHONY:	all clean setup test

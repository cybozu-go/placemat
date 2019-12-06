# Makefile for multi-host testing using ginkgo

# configuration variables
BRIDGE_ADDRESS = 10.0.0.1
BMC_ADDRESS = 10.1.0.1
NODE1 = 10.0.0.11
NODE2 = 10.0.0.12
POD1 = 10.0.0.21
POD2 = 10.0.0.22
BMC1 = 10.1.0.11
GINKGO = env GOFLAGS=-mod=vendor $(GOPATH)/bin/ginkgo
PLACEMAT_DATADIR = /var/scratch/placemat
export BRIDGE_ADDRESS NODE1 NODE2 POD1 POD2 BMC1 GINKGO

### for Go
GOFLAGS = -mod=vendor
export GOFLAGS

# non-configuration variables
SSH_PRIVKEY = $(realpath ./mtest_key)
OUTPUT := ./output
UBUNTU_IMAGE := ubuntu-18.04-server-cloudimg-amd64.img
PLACEMAT = $(abspath $(OUTPUT))/placemat
PMCTL = $(abspath $(OUTPUT))/pmctl
CLUSTER_YAML = $(abspath $(OUTPUT))/cluster.yml
EXAMPLE_CLUSTER_YAML = $(abspath $(OUTPUT))/cluster.example.yml
BMC_CERT = $(abspath $(OUTPUT))/bmc.cert
BMC_KEY = $(abspath $(OUTPUT))/bmc.key
SUITE = ./functions
ifeq ($(findstring /,$(SUITE)),)
  SUITE_PACKAGE = ./$(SUITE)
else
  SUITE_PACKAGE = $(SUITE)
endif

export SSH_PRIVKEY PLACEMAT PMCTL CLUSTER_YAML EXAMPLE_CLUSTER_YAML BMC_CERT BMC_KEY

TEST_DEPS = $(OUTPUT)/placemat \
	$(OUTPUT)/pmctl \
	$(OUTPUT)/cluster.yml \
	$(OUTPUT)/user-data_node1.yml \
	$(OUTPUT)/user-data_node2.yml \
	$(OUTPUT)/network1.yml \
	$(OUTPUT)/network2.yml \
	$(OUTPUT)/cluster.example.yml \
	$(OUTPUT)/network-config.example.yml \
	$(OUTPUT)/user-data.example.yml \
	$(OUTPUT)/bmc.cert \
	$(OUTPUT)/bmc.key \
	$(UBUNTU_IMAGE)

all:
	@echo "Specify one of these targets:"
	@echo
	@echo "    test     - run test."
	@echo "    setup    - install dependencies."
	@echo
	@echo "'make test' accepts SUITE."

$(UBUNTU_IMAGE):
	curl -sSLf -o $@ https://cloud-images.ubuntu.com/releases/18.04/release/$@

$(OUTPUT)/cluster.yml: cluster.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@BMC_ADDRESS@/$(BMC_ADDRESS)/' \
		-e 's/@POD1@/$(POD1)/' \
		-e 's/@POD2@/$(POD2)/' \
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

$(OUTPUT)/cluster.example.yml: ../examples/cluster.example.yml
	mkdir -p $(OUTPUT)
	sed -e 's/^url: .*/file: ..\/$(UBUNTU_IMAGE)/' $< > $@

$(OUTPUT)/network-config.example.yml: ../examples/network-config.example.yml
	mkdir -p $(OUTPUT)
	cp $< $@

$(OUTPUT)/user-data.example.yml: ../examples/user-data.example.yml
	mkdir -p $(OUTPUT)
	cp $< $@

$(OUTPUT)/bmc.cert $(OUTPUT)/bmc.key:
	openssl req -new -newkey rsa:4096 -days 3650 -nodes -x509 \
	-subj "/C=US/ST=Denial/L=Springfield/O=Dis/CN=*.www.example.com" \
	-keyout $(OUTPUT)/bmc.key  -out $(OUTPUT)/bmc.cert

$(OUTPUT)/placemat:
	mkdir -p $(OUTPUT)
	GOBIN=$(abspath $(OUTPUT)) go install ../pkg/$(notdir $@)

$(OUTPUT)/pmctl:
	mkdir -p $(OUTPUT)
	GOBIN=$(abspath $(OUTPUT)) go install ../pkg/$(notdir $@)

# Targets
test: $(TEST_DEPS)
	sudo rm -rf $(PLACEMAT_DATADIR)
	$(GINKGO) -v -focus="${TARGET}" $(SUITE_PACKAGE)

clean:
	rm -rf $(OUTPUT)

setup:
	go install github.com/onsi/ginkgo/ginkgo

.PHONY: all test clean setup

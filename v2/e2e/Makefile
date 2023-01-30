# Makefile for multi-host testing using ginkgo

# configuration variables
BRIDGE_ADDRESS = 10.0.0.1
BMC_ADDRESS = 10.1.0.1
NODE1 = 10.0.0.11
NODE2 = 10.0.0.12
NETNS1 = 10.0.0.21
NETNS2 = 10.0.0.22
BMC1 = 10.1.0.11
BMC2 = 10.1.0.12
PLACEMAT_DATADIR = /var/scratch/placemat
export BRIDGE_ADDRESS NODE1 NODE2 NETNS1 NETNS2 BMC1 BMC2

# non-configuration variables
SSH_PRIVKEY = $(realpath ./mtest_key)
OUTPUT := ./output
UBUNTU_IMAGE := ubuntu-22.04-server-cloudimg-amd64.img
PLACEMAT = $(abspath $(OUTPUT))/placemat2
PMCTL = $(abspath $(OUTPUT))/pmctl2
CLUSTER_YAML = $(abspath $(OUTPUT))/cluster.yml
EXAMPLE_CLUSTER_YAML = $(abspath $(OUTPUT))/cluster.example.yml

export SSH_PRIVKEY PLACEMAT PMCTL CLUSTER_YAML EXAMPLE_CLUSTER_YAML

TEST_DEPS = $(OUTPUT)/placemat2 \
	$(OUTPUT)/pmctl2 \
	$(OUTPUT)/cluster.yml \
	$(OUTPUT)/user-data_node1.yml \
	$(OUTPUT)/user-data_node2.yml \
	$(OUTPUT)/network1.yml \
	$(OUTPUT)/network2.yml \
	$(OUTPUT)/cluster.example.yml \
	$(OUTPUT)/network-config.example.yml \
	$(OUTPUT)/user-data.example.yml \
	$(UBUNTU_IMAGE)

all:
	@echo "Specify one of these targets:"
	@echo
	@echo "    test     - run test."
	@echo "    clean    - remove output directory."

$(UBUNTU_IMAGE):
	curl -sSLf -o $@ https://cloud-images.ubuntu.com/releases/22.04/release/$@

$(OUTPUT)/cluster.yml: cluster.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@BMC_ADDRESS@/$(BMC_ADDRESS)/' \
		-e 's/@NETNS1@/$(NETNS1)/' \
		-e 's/@NETNS2@/$(NETNS2)/' \
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

$(OUTPUT)/cluster.example.yml: ../../examples/cluster.example.yml
	mkdir -p $(OUTPUT)
	sed -e 's/^url: .*/file: ..\/$(UBUNTU_IMAGE)/' $< > $@

$(OUTPUT)/network-config.example.yml: ../../examples/network-config.example.yml
	mkdir -p $(OUTPUT)
	cp $< $@

$(OUTPUT)/user-data.example.yml: ../../examples/user-data.example.yml
	mkdir -p $(OUTPUT)
	cp $< $@

$(OUTPUT)/placemat2:
	mkdir -p $(OUTPUT)
	GOBIN=$(abspath $(OUTPUT)) go install ../cmd/$(notdir $@)

$(OUTPUT)/pmctl2:
	mkdir -p $(OUTPUT)
	GOBIN=$(abspath $(OUTPUT)) go install ../cmd/$(notdir $@)

.PHONY: test
test: $(TEST_DEPS)
	sudo rm -rf $(PLACEMAT_DATADIR)
	env E2ETEST=1 go test -failfast -count 1 -v . -args -ginkgo.progress -ginkgo.v

.PHONY: clean
clean:
	rm -rf $(OUTPUT)


SHELL=/usr/bin/env bash

all: build
.PHONY: all

BINS:=

basefee-monitor:
	go build -mod=vendor -o basefee-monitor ./cli/baseFee/
BINS+=basefee-monitor

sync-chain:
	go build -mod=vendor -o sync-chain ./synchain/
BINS+=sync-chain

basefee-auto:
	go build -o basefee-auto ./cli/baseFeeAuto/
BINS+=basefee-auto

power:
	go build -o power ./cli/pow
BINS+=power

build: $(BINS)

clean:
	rm -f $(BINS)

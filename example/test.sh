#!/bin/sh
go build -o example-server example.go && \
go build -o diamond-admin github.com/aerth/diamond && \
./example-server . && \
./diamond-admin -s diamond.socket


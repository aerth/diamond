#!/bin/sh
CGO_ENABLED=0 go build \
    -o diamond-admin \
    -ldflags="-s -w -X 'main.clientname=Diamond Admin'" \
    github.com/aerth/diamond/cmd/diamond-admin
#!/bin/bash
set -eou pipefail

GOOS=linux GOARCH=amd64 go build -o exec
zip allToAll.zip exec
../../bin/wsk -i action update allToAll --native allToAll.zip
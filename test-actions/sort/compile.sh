#!/bin/bash

GOOS=linux GOARCH=amd64 go build -o exec
zip assigner.zip exec
../../bin/wsk -i action update --native assignerGo assigner.zip
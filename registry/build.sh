#!/bin/bash

docker build -t 54.175.14.211:5000/whisk/registry .
docker push 54.175.14.211:5000/whisk/registry


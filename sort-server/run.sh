#!/bin/bash

docker run --rm -p 4343:4343 sort-server \
    -bucket test \
    -endpoint 172.17.0.1:3343 \
    -accessKeyID CMUKLLKXE6UER1NKPTEE \
    -secretAccessKey yGQXXwQ3o2PyGFSTazXUpMcNfIesG08omyH1+gTu \
    -registry 172.17.0.1:7777
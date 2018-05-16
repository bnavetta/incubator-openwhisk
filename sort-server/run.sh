#!/bin/bash

docker --host 34.228.208.174:4243 run \
	--name sort-server \
	--network whisknet \
	--rm -p 4343:4343 35.168.12.208:5000/sort-server \
    -bucket sorter \
    -endpoint s3.amazonaws.com \
    -accessKeyID AKIAIOSBKOGZKFWQWQQQ \
    -secretAccessKey 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 \
    -registry 10.0.13.4:7777

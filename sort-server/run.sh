#!/bin/bash

export DOCKER_HOST=52.90.40.79:4243

docker run \
	--name sort-server \
	--network default \
	-d --name sort-server \
	--rm -p 4343:4343 54.175.14.211:5000/sort-server \
    -bucket sorter \
    -endpoint s3.amazonaws.com \
    -accessKeyID AKIAIOSBKOGZKFWQWQQQ \
    -secretAccessKey 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 \
    -registry 10.0.87.6:7777

docker network connect whisknet sort-server

docker inspect sort-server

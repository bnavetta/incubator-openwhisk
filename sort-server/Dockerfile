FROM golang:1.10

RUN go get github.com/gorilla/mux
RUN go get github.com/minio/minio-go

COPY main.go /usr/src/main.go
COPY registry.go /usr/src/registry.go
WORKDIR /usr/src

RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o sort-server -v

# This will start a multi-stage Docker build
# The first container, with Go, will build the sort server
# The next, which has nothing, only has our executable
#FROM scratch

FROM alpine

COPY --from=0 /usr/src/sort-server /usr/local/bin/sort-server
ENTRYPOINT [ "/usr/local/bin/sort-server" ]

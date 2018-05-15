package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	minio "github.com/minio/minio-go"
)

var minioClient *minio.Client
var bucketName string

type PartitionUpload struct {
	doneCh chan error
	reader *io.PipeReader
	writer *io.PipeWriter
}

func startPartitionUpload(path string) PartitionUpload {
	reader, writer := io.Pipe()
	doneCh := make(chan error)

	go func() {
		defer reader.Close()
		log.Printf("Uploading partition %v", path)
		_, err := minioClient.PutObject(bucketName, path, reader, -1, minio.PutObjectOptions{
			ContentType: "text/plain",
		})
		if err != nil {
			fmt.Printf("Error uploading partition %v: %v", path, err)
		}
		doneCh <- err
	}()

	return PartitionUpload{reader: reader, writer: writer, doneCh: doneCh}
}

func UploadDatasetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dataset := vars["dataset"]
	partitions, err := strconv.ParseInt(vars["partitions"], 10, 32)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid partition count: %v", err), 400)
		return
	}

	uploads := make([]PartitionUpload, partitions)
	for i := int64(0); i < partitions; i++ {
		uploads[i] = startPartitionUpload(fmt.Sprintf("%v/input-partitions/%v", dataset, i))
		defer uploads[i].writer.Close()
	}

	lines := 0
	src := bufio.NewScanner(r.Body)
	for src.Scan() {
		line := src.Bytes()
		_, err = uploads[lines%int(partitions)].writer.Write(line)
		_, err = uploads[lines%int(partitions)].writer.Write([]byte("\n"))
		if err != nil {
			http.Error(w, fmt.Sprintf("Error writing partition: %v", err), 500)
			return
		}
		lines++
	}

	if err = src.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error reading input data: %v", err), 500)
		return
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, "Created %v partitions for %v", partitions, dataset)
}

func GetPartitionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := fmt.Sprintf("%v/input-partitions/%v", vars["dataset"], vars["id"])

	obj, err := minioClient.GetObject(bucketName, path, minio.GetObjectOptions{})
	if err != nil {
		log.Printf("Error getting object %v: %v", path, err)

		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, err.Error())
		return
	}

	info, err := obj.Stat()
	if err != nil {
		log.Printf("Error getting object information for partition %v: %v", path, err)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, err.Error())
		return
	}
	log.Printf("Serving partition %v with object %v", path, info)

	w.Header().Set("Content-Type", info.ContentType)
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, obj)
	if err != nil {
		log.Printf("Error transferring partition %v: %v", path, err)
	}
}

func UploadSplitHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dataset := vars["dataset"]
	split := vars["split"]
	path := fmt.Sprintf("%v/output-splits/%v", dataset, split)
	log.Printf("Writing split %v of dataset %v to %v", split, dataset, path)

	defer r.Body.Close()
	_, err := minioClient.PutObject(bucketName, path, r.Body, -1, minio.PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		fmt.Printf("Error uploading output split to %v: %v", path, err)
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, "Wrote output split %v", path)
}

func ComposeOutputHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dataset := vars["dataset"]
	log.Printf("Composing output for dataset %v", dataset)

	srcs := make([]minio.SourceInfo, 0)
	for split := '0'; split < 'Z'; split++ {
		path := fmt.Sprintf("%v/output-splits/%c", dataset, split)

		_, err := minioClient.StatObject(bucketName, path, minio.StatObjectOptions{})
		if err != nil {
			// Check if it's just a missing split - there might not have been sort entries for each key
			errResp := err.(minio.ErrorResponse)
			if errResp.Code == "NoSuchKey" {
				log.Printf("Dataset output %v is missing split '%c'", dataset, split)
				continue
			} else {
				log.Printf("Error checking status of split %v: %v", path, errResp)
				http.Error(w, err.Error(), 500)
				return
			}
		}

		sourceInfo := minio.NewSourceInfo(bucketName, path, nil)
		srcs = append(srcs, sourceInfo)
	}

	destPath := fmt.Sprintf("%v/output", dataset)
	dst, err := minio.NewDestinationInfo(bucketName, destPath, nil, nil)
	if err != nil {
		log.Printf("Error creating info for composed destination %v: %v", destPath, err)
		http.Error(w, err.Error(), 500)
		return
	}

	err = minioClient.ComposeObject(dst, srcs)
	if err != nil {
		log.Printf("Error composing output splits: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}

	params := make(url.Values)
	params.Set("response-content-type", "text/plain")
	presignedUrl, err := minioClient.PresignedGetObject(bucketName, destPath, 7*24*time.Hour, params)
	if err != nil {
		log.Printf("Unable to generate presigned URL for composed output %v: %v", destPath, err)
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Location", presignedUrl.String())
	w.WriteHeader(http.StatusPermanentRedirect)
	fmt.Fprintf(w, "Output available at %v", presignedUrl)
}

func main() {
	var endpoint string
	var accessKeyID string
	var secretAccessKey string
	var port int
	var registryAddr string

	flag.StringVar(&registryAddr, "registry", "", "Registry server address")
	flag.StringVar(&bucketName, "bucket", "wsk", "Minio bucket name")
	flag.IntVar(&port, "port", 4343, "Port to listen on")
	flag.StringVar(&endpoint, "endpoint", "localhost:9000", "Endpoint of the Minio server")
	flag.StringVar(&accessKeyID, "accessKeyID", "", "Access key ID for Minio")
	flag.StringVar(&secretAccessKey, "secretAccessKey", "", "Secret access key for Minio")
	flag.Parse()

	var err error
	minioClient, err = minio.New(endpoint, accessKeyID, secretAccessKey, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to Minio: %v\n", err)
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.HandleFunc("/{dataset}/partitions/{id}", GetPartitionHandler).Methods("GET")
	r.HandleFunc("/{dataset}", UploadDatasetHandler).Methods("PUT").Queries("partitions", "{partitions}")
	r.HandleFunc("/{dataset}/outputs/{split}", UploadSplitHandler).Methods("PUT")
	r.HandleFunc("/{dataset}/output", ComposeOutputHandler).Methods("GET")

	registry := NewRegistry(registryAddr)
	err = registry.Register("sort-file-store")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not register: %v", err)
		os.Exit(1)
	}

	http.Handle("/", r)
	http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
}

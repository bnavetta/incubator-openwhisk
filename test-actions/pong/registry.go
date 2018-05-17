package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
)

type Registry struct {
	address string
}

func NewRegistry(address string) Registry {
	fmt.Printf("Using registry at %v\n", address)
	return Registry{address: address}
}

func (r *Registry) Register(name string, address string) error {
	fmt.Printf("Registering for service %v\n", name)

	conn, err := net.Dial("tcp", r.address)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "r %v %v", name, address)
	return err
}

func (r *Registry) Lookup(name string) (string, error) {
	conn, err := net.Dial("tcp", r.address)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "g %v", name)
	if err != nil {
		return "", err
	}

	response, err := ioutil.ReadAll(conn)
	if err != nil {
		return "", err
	}

	parts := bytes.Split(response, []byte(" "))
	if len(parts) != 2 {
		return "", errors.New("Malformed response")
	}

	addr := string(parts[0])
	if addr == "Invalid" {
		return "", errors.New("Undefined name")
	}

	fmt.Printf("Looked up %v for service %v\n", addr, name)

	return addr, nil
}

func (r *Registry) LookupBlocking(name string) (string, error) {
	for {
		addr, err := r.Lookup(name)
		if err == nil {
			return addr, nil
		} else if err.Error() != "Undefined name" {
			return "", err
		}
	}
}

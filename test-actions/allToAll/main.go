package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
)

type Registry struct {
	address string
}

func NewRegistry(address string) Registry {
	return Registry{address: address}
}

func (r *Registry) Register(name string) error {
	fmt.Printf("Registering for service %v\n", name)

	conn, err := net.Dial("tcp", r.address)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "r %v", name)
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

type AllToAll struct {
	id           int
	myNumber     int
	received     []int
	instances    int
	listenerDone chan bool
}

func NewAllToAll(params Parameters) AllToAll {
	return AllToAll{
		id:           params.Id,
		myNumber:     params.MyNumber,
		instances:    params.Instances,
		received:     make([]int, 0),
		listenerDone: make(chan bool, 1),
	}
}

func (a *AllToAll) Register(registry Registry) error {
	return registry.Register(fmt.Sprintf("a2a-%v", a.id))
}

func (a *AllToAll) Listen(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	for len(a.received) < a.instances-1 {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}

		data, err := ioutil.ReadAll(conn)
		if err != nil {
			return err
		}

		num := binary.BigEndian.Uint32(data)
		fmt.Printf("Received %v from %v\n", num, conn.RemoteAddr())
		a.received = append(a.received, int(num))
		conn.Close()
	}

	a.listenerDone <- true

	return nil
}

func (a *AllToAll) SendValue(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	fmt.Printf("Sending value to %v\n", conn.RemoteAddr())

	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(a.myNumber))

	err = binary.Write(conn, binary.BigEndian, uint32(a.myNumber))

	return err
}

func (a *AllToAll) SendAll(registry Registry) error {
	for id := 0; id < a.instances; id++ {
		if id == a.id {
			continue
		}

		fmt.Printf("Will send to instance %v\n", id)

		addr, err := registry.LookupBlocking(fmt.Sprintf("a2a-%v", id))
		if err != nil {
			return err
		}

		err = a.SendValue(addr + ":9898")
		if err != nil {
			return err
		}
	}

	return nil
}

func jsonify(val interface{}) string {
	res, _ := json.Marshal(val)
	return string(res)
}

func logError(err error) {
	fmt.Println(jsonify(map[string]string{"error": err.Error()}))
}

type Parameters struct {
	Registry  string
	Id        int
	MyNumber  int
	Instances int
}

func main() {
	arg := os.Args[1]
	var params Parameters
	json.Unmarshal([]byte(arg), &params)

	fmt.Printf("My ID: %v, my number: %v\n", params.Id, params.MyNumber)

	registry := NewRegistry(params.Registry)

	a2a := NewAllToAll(params)

	err := a2a.Register(registry)
	if err != nil {
		logError(err)
		return
	}

	go a2a.Listen(":9898")
	err = a2a.SendAll(registry)
	if err != nil {
		logError(err)
		return
	}

	<-a2a.listenerDone

	sum := a2a.myNumber
	for _, num := range a2a.received {
		sum += num
	}

	fmt.Println(jsonify(map[string]interface{}{
		"sum":      sum,
		"myNumber": a2a.myNumber,
		"received": a2a.received,
	}))
}

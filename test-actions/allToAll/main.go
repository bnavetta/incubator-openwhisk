package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"
)

func getOverlayIP() string {
	// TODO: assuming eth1 might be too fragile, could do a subnet check
	iface, err := net.InterfaceByName("eth1")
	if err != nil {
		return err.Error()
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return err.Error()
	}

	for _, addr := range addrs {
		switch a := addr.(type) {
		case *net.IPAddr:
			return a.IP.String()
		case *net.IPNet:
			return a.IP.String()
		default:
			continue
		}
	}

	return ""
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

func (a *AllToAll) Listen(registry *Registry) error {
	addr := getOverlayIP() + ":9898"

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	err = registry.Register(fmt.Sprintf("a2a-%v", a.id), getOverlayIP())
	if err != nil {
		logError(err)
		return err
	}

	for len(a.received) < a.instances-1 {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}

		data, err := ioutil.ReadAll(conn)
		if err != nil {
			logError(err)
			return err
		}

		num := binary.BigEndian.Uint32(data)
		fmt.Printf("Received %v from %v\n", num, conn.RemoteAddr())
		a.received = append(a.received, int(num))
		conn.Close()
	}

	fmt.Printf("Received all values! %v\n", a.received)

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
	completions := make(chan error, a.instances)

	for id := 0; id < a.instances; id++ {
		if id == a.id {
			continue
		}

		go func(id int) {
			fmt.Printf("Will send to instance %v\n", id)

			addr, err := registry.LookupBlocking(fmt.Sprintf("a2a-%v", id))
			if err != nil {
				completions <- err
				return
			}

			fmt.Printf("Looked up %v for id %v\n", addr, id)

			err = a.SendValue(addr + ":9898")
			completions <- err
		}(id)
	}

	for i := 0; i < a.instances-1; i++ {
		res := <-completions
		if res != nil {
			return res
		}
	}

	fmt.Println("Sent to all other lambdas!")

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
	go func() {
		time.Sleep(55 * time.Second)
		fmt.Println("{ \"status\": \"timed out\" }")
		os.Exit(0)
	}()

	arg := os.Args[1]
	var params Parameters
	json.Unmarshal([]byte(arg), &params)

	fmt.Printf("My ID: %v, my number: %v\n", params.Id, params.MyNumber)

	registry := NewRegistry(params.Registry)

	a2a := NewAllToAll(params)

	go a2a.Listen(&registry)
	err := a2a.SendAll(registry)
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

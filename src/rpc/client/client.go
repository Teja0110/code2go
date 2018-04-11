package main

import (
	"errors"
	"net/rpc"
	"log"
	"fmt"
)

type Args struct {
	A, B int
}

type Quotient struct {
	Quo, Rem int
}

type Arith int

func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

func (t *Arith) Divide(args *Args, quo *Quotient) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	quo.Quo = args.A / args.B
	quo.Rem = args.A % args.B
	return nil
}

func main() {
	serverAddress := "127.0.0.1"
	client, err := rpc.DialHTTP("udp", serverAddress + ":1234")
	if err != nil {
		log.Fatal("Fatal error:", err)
	}
	//args := &server.Args{7, 8}
	var args Args = Args{7, 8}
	var reply int
	err = client.Call("Arith.Multiply", args, &reply)
	if err != nil {
		log.Fatal("arith error:", err)
	}
	fmt.Printf("Arith: %d * %d = %d", args.A, args.B, reply)
}

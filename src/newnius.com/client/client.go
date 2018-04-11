package main

import (
	"fmt"
	"log"
	"time"
	pb "newnius.com/ntp"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	address = "ntp.newnius.com:8844"
)

var delay time.Duration

func getTime() time.Time {
	return time.Now().Add(delay)
}

func tick () {
	for {
		t := getTime()
		fmt.Printf("Current: %v/%v %02d:%02d:%02d\n", int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second());
		time.Sleep(1000 * time.Millisecond)
	}
}

func sync() {
	creds, err := credentials.NewClientTLSFromFile("server-cert.pem", "ntp.newnius.com")
	if err != nil {
		log.Fatalf("cert load error: %s", err)
	}

	//conn, err := grpc.Dial(address, grpc.WithInsecure())
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("unable to connect: %v", err)
	}
	defer conn.Close()

	var c pb.NtpClient

	for {
		time.Sleep(5000 * time.Millisecond)
		c = pb.NewNtpClient(conn)
		start := time.Now().UnixNano()
		//time.Sleep(5000 * time.Millisecond)
		r, err := c.Query(context.Background(), &pb.NtpRequest{})
		//time.Sleep(5000 * time.Millisecond)
		if err != nil {
			fmt.Printf("WARNING: unable to sync: %v\n", err)
			continue
		}
		//fmt.Printf("Greeting: %v %s\n", i, r.Message)
		//current = t.Add(time.Since(start) / 2)
		//fmt.Println(r.Message)
		m := ( time.Now().UnixNano() - start ) / 2 ;// / int64(time.Millisecond);
		//fmt.Println(time.Duration(m))
		delay = time.Duration(r.Message - start - m)
		//fmt.Println(delay)
		fmt.Printf("RPC call cost %v\n", time.Duration(m * 2))
		//fmt.Printf("RPC call cost %v\n", time.Since(start))
		//fmt.Printf("call cost %v\n", int64(time.Since(start) / time.Microsecond))
		fmt.Println("DEBUG: synced")
	}
}

func main() {
	go sync()
	tick()
}

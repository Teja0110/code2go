<center><h1>Network Time Synchronization based on RPC</h1></center>
<center><strong>Shuai Liu, MG1733039</strong></center>
<hr/>

# Abstract
It is vital in some cases to sync time with server. This expriment uses RPC call to realize network time synchronization.

## keywords
RPC, Time Synchronization, Go

# Introduction
There are many cases that time synchonization is more or less significant. For some competitive online games, it would be a mess if various clients hold inconsistant time. In this expriment, we try to design and realize a network time syncronization in the architecture of C/S.

We have a plenty of communication methods between server and clients to choose from. Considering convenient and scalability, and finally RPC is selected. Mature RPC fragments have the advantages of service discovery, retry on failure, load balance which is rather helpful. As to the develop language, go is choosen since it is designed for such high concurrency distributed appliations.


# Design & Realization
There are several requirements from this network time syncronization application. First of all, it should be language independent and cross-platform which means the server is able to support different kinds of clients running in various platforms. Secondly, the server can serve a quantity of clients at the same time. Thirdly,  in this application, only authorized clients could get time from server. Except for the requirements above, we have more things to take into consideration such as network delay, network failure, scalability, etc.

After research, we found some available packages of RPC in go language and after comperison gRPC turned out to be the best choice from our application.

||rpc|rpc/http|rpcx|grpc|thrift|
|---|---|---|---|---|---|
|Language independent | N | N | N | Y | Y |
|Authorization| N | N | Y | Y | N |

## Communication delay & failure
It is rather common for an online application that the network latency is too high to be ignored and even package lose. This problem is more obvious for our network time synchonization application. However, it is not easy to solve this problem since the network changes unpredictedly all the time. In our expriment, we don't want to have a extremely precise result and can tolerate a second level of error.

To do this, we record the local time before and after making RPC call which we call `start` and `stop`, and obviously `stop - start` is the time need for one RPC call. We assume that processing time in both sides can be ignored and the network changes little during the request. So we can calculate the time to reach another side simply by dividing the difference by 2.

```go
start := time.Now().UnixNano()

// do RPC call

stop := time.Now().UnixNano()

delay := ( stop - start ) / 2;
```

Then, we can get the real time by subtracting delay from the time replied.

```go
realtime = reply.time - delay
```

## Setup Environment

## Install Go
```bash
# Download
wget https://storage.googleapis.com/golang/go1.9.1.linux-amd64.tar.gz

# Extract
sudo tar -C /usr/local/ -xzf go1.9.1.linux-amd64.tar.gz

# Configure system environments of go
sudo bash -c 'cat >> /etc/profile <<EOF
export GOPATH=/home/newnius/go
export PATH=\$PATH:/usr/local/go/bin
EOF'

# Update environment variables
source /etc/profile
```

## Install Protobuf
`Protobuf` is a serialization method and is adopted by gRPC in message exchange.

```bash
# Download
sudo apt install autoconf automake libtool g++

# Make & Install protobuf
git clone https://github.com/google/protobuf.git

cd protobuf/

./autogen.sh

./configure

make

sudo make install

sudo ldconfig

# Configure system environments of go
sudo bash -c 'cat >> /etc/profile <<EOF
export PATH=\$PATH:\$GOPATH/bin
EOF'

# Update environment variables
source /etc/profile
```

## Create Protocol Buffers IDL

ntp/ntp.proto
```
syntax = "proto3";

option java_package = "com.colobu.rpctest";

package ntp;

service Ntp {
	rpc Query (NtpRequest) returns (NtpReply) {}
}

message NtpRequest {
}

message NtpReply {
	int64 message = 1;
}
```

## Generate pb files
Here we generates two pb files since we will also create a python client as well.

Get gRPC package and protobuf-go plugin package
```bash
go get -u google.golang.org/grpc

go get -u github.com/golang/protobuf/{proto,protoc-gen-go}

protoc -I ntp ntp/ntp.proto --go_out=plugins=grpc:ntp
```

Get gRPC package and protobuf plugin-python package
```bash
sudo apt install python-pip

python /usr/bin/pip install grpcio

python /usr/bin/pip install grpcio-tools

python -m grpc_tools.protoc -I go/src/newnius.com/ntp --python_out=. --grpc_python_out=. go/src/newnius.com/ntp/ntp.proto
```

## Authorization
Although we can pass token every time we make RPC call, it is not a good design. Luckily, gRPC provides the ability to verify clients (tls). We generates a key file along with a certificate file, RPC call from unauthorized clients without valid certificate file will be rejected by the gRPC fragment automatically. What't more, gRPC can also distinguish each client by setting a unique token, we skipped the token verification since the certificate files is secure enough for our case.

With the terribe network environment where full of monitors and sniffers, it is significant to encrypt messages in network. After research, we find gRPC would encrypt the messages if tls is used, thus we don't have to trouble ourselves.

Now, we issue certificate files to `ntp.newnius.com` and use them for authorization later.

```bash
openssl req -x509 -newkey rsa:4096 -keyout server-key.pem -out server-cert.pem -days 365 -nodes -subj '/CN=ntp.newnius.com'
```

## Server

### server.go
```go
package main

import (
	"log"
	"net"
	"time"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "newnius.com/ntp"
	"google.golang.org/grpc/credentials"
)

var (
	certFile = "server-cert.pem"
	keyFile = "server-key.pem"
)

const (
	port = ":8844"
)

type server struct{}

func (s *server) Query(ctx context.Context, in *pb.NtpRequest) (*pb.NtpReply, error) {
	// we add another 100 seconds to simulate a time delay behind
	return &pb.NtpReply{Message: time.Now().UnixNano() + 100000000000}, nil
}

func main() {
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		log.Fatalf("Failed to setup tls: %v", err)
	}
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.Creds(creds),
	)
	pb.RegisterNtpServer(s, &server{})
	s.Serve(lis)
}
```
## Client (in Go and Python)
We realized two version clients in go and python.

The client works as follows:
  - Start a new thread to make RPC calls in a certain frequency and update the global variable `delay` which means the time delay between server and client (network delay included)
  - Main thread displays current time every second. The outputed time is calculated by `now = localtime + delay`

Notice that we didn't display time in digital mode or analogue mode since they are not the main purpose for this expriment and GUI programming is so time-consuming. Instead, we just print current time every second.

### client.go
```go
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
		r, err := c.Query(context.Background(), &pb.NtpRequest{})
		if err != nil {
			fmt.Printf("WARNING: unable to sync: %v\n", err)
			continue
		}
		m := ( time.Now().UnixNano() - start ) / 2 ;// / int64(time.Millisecond);
		delay = time.Duration(r.Message - start - m)
		fmt.Printf("RPC call cost %v\n", time.Duration(m * 2))
		fmt.Println("DEBUG: synced")
	}
}

func main() {
	go sync()
	tick()
}
```

### ntp.py
```python
# -*- coding: utf-8 -*-
import grpc
import ntp_pb2
import datetime, time, threading

import ntp
ntp.delay = 0.0

running = True

def sync():
  creds = grpc.ssl_channel_credentials(open('server-cert.pem').read())
  channel = grpc.secure_channel('ntp.newnius.com:8844', creds)
  stub = ntp_pb2.NtpStub(channel)

  request = ntp_pb2.NtpRequest()
  while running:
    start = round(time.time() * 1000)
    reply = stub.Query(request)
    m = ( round(time.time() * 1000) - start ) / 2
    print "RPC call used: ", m * 2, "ms"
    ntp.delay = ( reply.message / float(1000000) - start - m ) / float(1000)
    print "DEBUG: synced"
    time.sleep(5)

def tick():
  while True:
    current = datetime.datetime.now() + datetime.timedelta(0, ntp.delay)
    print "Current: ", current
    time.sleep(1)

if __name__ == '__main__':
  try:
    #sync()
    t = threading.Thread(target=sync)
    t.start()
    tick()
  except KeyboardInterrupt:
    running = False
```

# Validation
Several tests were made for different purposes.

## Correctness & Precision

We deployed the server in a remote VPS whose average `ping` latency from the develop machine is around 400ms and made following settings in server / client:

- Add the time replied by server by 100 seconds to simulate a situation that the client is 100s behind server
- Sleep for 5s before & after RPC call to generate a longer network delay

The result shows that the client successfully calculated the real time fixed by delay and retry on failure.

<img alt="Correctness" src="figures/ntp-correctness.png" style="zoom:60%" />

<img alt="Fault Tolerance" src="figures/ntp-fault-tolerance.png" style="zoom:60%" />

## Cross-platform & Language-independent

Since we don't have Windows system in hand, so all the work were made in Linux. But consider that python is a cross-platform language so we can say that the client can run on most platforms.

<img alt="Language-independent" src="figures/ntp-python.png" style="zoom:60%" />

## Authorization & Security
In our tests, only clients using the right method and certificate files can get response from server.

<img alt="Security" src="figures/ntp-security.png" style="zoom:60%" />

## Concurrency Performance
We deployed 6 VPSs to perform as clients and one as server. The resource of VPSs ranges from 1 Core/0.5G Ram to 1 Core/2G Ram, and the server is deployed in a 1 core/2G Ram VPS. All of them are in system Linux without desktop and most of the resources are vacant. To better simulate the real environment, the VPSs are located around the world.

First of all, we tried to run as many as possible clients in the 6 nodes, each client would send a query requst every 5 seconds. The total number of client is about 2000.

<img alt="Client Numbers" src="figures/ntp-client-cnt.png" style="zoom:60%" />

<img alt="Client Numbers" src="figures/ntp-client-cnt-2.png" style="zoom:60%" />

After that, we run a client in our develop machine, and the log showed that the RPC call is made successfully. The figures below shows that under the concurrency of 1000, the server still works perfect and can hold more clients.

Establised connections
<br>
<img alt="Server load" src="figures/ntp-server-cnt.png" style="zoom:60%" />

Server load
<br>
<img alt="Server load" src="figures/ntp-server-load.png" style="zoom:60%" />

# Future Work
There are still a lot of work to do with this application.

Although the communication delay is considered, it is not that precise for higher requirements. To achieve this goal, we can apply more approaches to fix this with more than one query. Secondly, in linux system, each process can only have at most 1024 active connections, which means only 1024 clients can be served as the same time. What's more, each connection may consume more resource than needed. These limitations can be solved by changing the settings.

What's more, gRPC uses `HTTP`, which is based on `TCP` in communication, this improves performance for frequent queries. But in our case, the request may be very slow and failure is acceptable, so we can use UDP to serve more clients.

# Conclusion
This expriment shows that RPC is a good fragment to make communications between server and client. With the help of a muture fragement, we can easily build a high available online application.

# References

[1] [Getting Started - The Go Programming Language](https://golang.org/doc/install)

[2] [c - Error on trying to run a simple RPC program - Stack Overflow](https://stackoverflow.com/questions/10448696/error-on-trying-to-run-a-simple-rpc-program)

[3] [RPC: Unknown Protocol - Stack Overflow](https://stackoverflow.com/questions/3818624/rpc-unknown-protocol)

[4] [Go官方库RPC开发指南](http://colobu.com/2016/09/18/go-net-rpc-guide/)

[5] [golang与java间的json-rpc跨语言调用](http://www.cnblogs.com/geomantic/p/4751859.html)

[6] [Go语言内部rpc简单实例,实现python调用go的jsonrpc小实例](http://blog.csdn.net/fyxichen/article/details/46998101)

[7] [GO语言RPC调用方案调研](https://scguoi.github.io/DivisionByZero/2016/11/15/GO%E8%AF%AD%E8%A8%80RPC%E6%96%B9%E6%A1%88%E8%B0%83%E7%A0%94.html)

[8] [gRPC 初探 - phper成长之路 - SegmentFault](https://segmentfault.com/a/1190000011483346)

[9] [Google 开源 RPC 框架 gRPC 初探](http://blog.jrwang.me/2016/grpc-at-first-view/)

[10] [Authentication in gRPC](http://mycodesmells.com/post/authentication-in-grpc)

[11] [Golang gRPC实践 连载四 gRPC认证](https://my.oschina.net/ifraincoat/blog/879545)

[12] [Secure gRPC with TLS/SSL · Libelli](https://bbengfort.github.io/programmer/2017/03/03/secure-grpc.html)

[13] [开源RPC（gRPC/Thrift）框架性能评测](http://www.eit.name/blog/read.php?566)

[14] [Go RPC 开发指南](https://smallnest.gitbooks.io/go-rpc/content/)

[15] [grpc / Authentication](https://grpc.io/docs/guides/auth.html)

[16] [Unix系统下的连接数限制](http://www.jianshu.com/p/faccda312cfe)

[17] [Linux下高并发socket最大连接数所受的各种限制](http://blog.sae.sina.com.cn/archives/1988)

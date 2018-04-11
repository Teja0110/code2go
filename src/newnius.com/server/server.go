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
	//loc, err := time.LoadLocation(in.Timezone)
	//if err != nil {
	//	zone, _ := time.Now().Zone()
	//	loc, _ = time.LoadLocation(zone)
	//}
	//set timezone,  
	//now := time.Now().In(loc)
	return &pb.NtpReply{Message: time.Now().UnixNano() + 100000000000}, nil
}

func main() {
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		log.Fatalf("Failed to setup tls: %v", err)
	}
	//log.Println(creds)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.Creds(creds),
	)
	pb.RegisterNtpServer(s, &server{})
	//reflection.Register(s)
	//if err := s.Serve(lis); err != nil {
	//	log.Fatal("failed to serve: %v", err)
	//}
	s.Serve(lis)
}

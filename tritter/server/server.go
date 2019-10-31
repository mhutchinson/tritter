package main

import (
	"context"
	"flag"
	"net"

	"github.com/golang/glog"
	"github.com/mhutchinson/tritter/tritter"
	"google.golang.org/grpc"
)

const (
	listenAddr = ":50051"
)

// server is used to implement TritterServer.
type server struct {
	tritter.UnimplementedTritterServer
}

// Send implements TritterServer.Send.
func (s *server) Send(ctx context.Context, in *tritter.SendRequest) (*tritter.SendResponse, error) {
	glog.Infof("Send: %v", in.GetMessage())
	return &tritter.SendResponse{}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	tritter.RegisterTritterServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		glog.Fatalf("failed to serve: %v", err)
	}
}

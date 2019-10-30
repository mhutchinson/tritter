package main

import (
	"context"
	"flag"
	"net"

	"github.com/golang/glog"
	pb "github.com/mhutchinson/tritter"
	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

// server is used to implement TritterServer.
type server struct {
	pb.UnimplementedTritterServer
}

// Send implements TritterServer.Send.
func (s *server) Send(ctx context.Context, in *pb.SendRequest) (*pb.SendResponse, error) {
	glog.Infof("Send: %v", in.GetMessage())
	return &pb.SendResponse{}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", port)
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterTritterServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		glog.Fatalf("failed to serve: %v", err)
	}
}

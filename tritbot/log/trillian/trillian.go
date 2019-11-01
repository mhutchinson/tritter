package main

import (
	"context"
	"flag"
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/google/trillian"
	"github.com/google/trillian/client"
	tt "github.com/google/trillian/types"
	"github.com/mhutchinson/tritter/tritbot/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	_ "github.com/google/trillian/merkle/rfc6962" // Load hashers
)

const (
	listenAddr = "localhost:50053"
)

var (
	logAddr        = flag.String("log_addr", "localhost:50054", "TCP address of Trillian log/admin server")
	connectTimeout = flag.Duration("connect_timeout", time.Second, "the timeout for connecting to the backend")

	treeID = flag.Int64("tree_id", -1, "the tree ID of the log to use")
)

type trillianLogger struct {
	log.UnimplementedLoggerServer

	c  *client.LogClient
	lc *grpc.ClientConn // Close this after use.
}

// newTrillianLogger creates a trillianLogger from the flags.
func newTrillianLogger() *trillianLogger {
	if *treeID <= 0 {
		glog.Fatalf("tree_id must be provided and positive, got %d", *treeID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *connectTimeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, *logAddr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		glog.Fatalf("did not connect: %v", err)
	}
	admin := trillian.NewTrillianAdminClient(conn)
	tree, err := admin.GetTree(ctx, &trillian.GetTreeRequest{TreeId: *treeID})
	if err != nil {
		glog.Fatalf("failed to get tree: %v", err)
	}
	v, err := client.NewLogVerifierFromTree(tree)
	if err != nil {
		glog.Fatalf("failed to create verifier from tree: %v", err)
	}

	log := trillian.NewTrillianLogClient(conn)
	c := client.New(*treeID, log, v, tt.LogRootV1{})

	return &trillianLogger{
		c:  c,
		lc: conn,
	}
}

// Log implements log.LoggerServer.Log.
func (l *trillianLogger) Log(ctx context.Context, in *log.LogRequest) (*log.LogResponse, error) {
	msg := in.GetMessage()
	if len(msg.GetMessage()) == 0 || len(msg.GetUser()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Message and user required")
	}

	bs, err := proto.Marshal(msg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal log message: %v", err)
	}
	l.c.AddLeaf(ctx, bs)
	r := l.c.GetRoot()
	glog.Infof("Logged to Trillian and included in r=%d: %v", r.Revision, msg)

	return &log.LogResponse{}, nil
}

func (l *trillianLogger) close() error {
	return l.lc.Close()
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	l := newTrillianLogger()
	defer l.close()
	log.RegisterLoggerServer(s, l)
	glog.Infof("Serving trillian logger on %v", listenAddr)
	if err := s.Serve(lis); err != nil {
		glog.Fatalf("failed to serve: %v", err)
	}
}

// A proxy that sends things on to tritter and logs the requests.
package main

import (
	"context"
	"flag"
	"time"

	"github.com/golang/glog"
	pb "github.com/mhutchinson/tritter"
	"google.golang.org/grpc"
)

var (
	tritterAddr    = flag.String("tritter_addr", "localhost:50051", "the address of the tritter server")
	connectTimeout = flag.Duration("connect_timeout", time.Second, "the timeout for connecting to the server")
	sendTimeout    = flag.Duration("send_timeout", 100*time.Millisecond, "the timeout for sending each message")
)

type tritBot struct {
	c       pb.TritterClient
	timeout time.Duration
}

func (t *tritBot) Send(ctx context.Context, msg string) error {
	glog.Infof("Sending message '%v'", msg)

	// TODO(mhutchinson): log the message

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	_, err := t.c.Send(ctx, &pb.SendRequest{Message: msg})
	return err
}

func main() {
	flag.Parse()

	// Read the message from the argument list
	if len(flag.Args()) == 0 {
		glog.Fatal("Required arguments: messages to send")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *connectTimeout)
	defer cancel()

	// Set up a connection to the server.
	conn, err := grpc.DialContext(ctx, *tritterAddr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		glog.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	t := tritBot{
		c:       pb.NewTritterClient(conn),
		timeout: *sendTimeout,
	}

	for _, msg := range flag.Args() {
		if err := t.Send(context.Background(), msg); err != nil {
			glog.Fatalf("could not greet: %v", err)
		}
	}
	glog.Infof("Successfully sent %d messages", len(flag.Args()))
}

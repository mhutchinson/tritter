// A proxy that sends things on to tritter and logs the requests.
package main

import (
	"context"
	"flag"
	"os/user"
	"time"

	"github.com/golang/glog"
	"github.com/mhutchinson/tritter/tritbot/log"
	"github.com/mhutchinson/tritter/tritter"
	"google.golang.org/grpc"
)

var (
	tritterAddr    = flag.String("tritter_addr", "localhost:50051", "the address of the tritter server")
	connectTimeout = flag.Duration("connect_timeout", time.Second, "the timeout for connecting to the server")
	sendTimeout    = flag.Duration("send_timeout", 500*time.Millisecond, "the timeout for logging & sending each message")

	loggerAddr = flag.String("logger_addr", "localhost:50052", "the address of the logger server")
)

type tritBot struct {
	c       tritter.TritterClient
	timeout time.Duration

	log log.LoggerClient
}

func (t *tritBot) Send(ctx context.Context, msg log.InternalMessage) error {
	ctx, cancel := context.WithTimeout(ctx, *sendTimeout)
	defer cancel()

	// First write the message to the log.
	if _, err := t.log.Log(ctx, &log.LogRequest{Message: &msg}); err != nil {
		return err
	}

	// Second: check the message is in the log.

	// Then continue to send the message to the server.
	_, err := t.c.Send(ctx, &tritter.SendRequest{Message: msg.GetMessage()})
	return err
}

func main() {
	flag.Parse()

	// Read the message from the argument list.
	if len(flag.Args()) == 0 {
		glog.Fatal("Required arguments: messages to send")
	}

	user, err := user.Current()
	if err != nil {
		glog.Fatalf("could not determine user: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *connectTimeout)
	defer cancel()

	// Set up a connection to the Tritter server.
	tCon, err := grpc.DialContext(ctx, *tritterAddr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		glog.Fatalf("did not connect: %v", err)
	}
	defer tCon.Close()

	// Set up a connection to the Logger server.
	lCon, err := grpc.DialContext(ctx, *loggerAddr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		glog.Fatalf("did not connect: %v", err)
	}
	defer lCon.Close()

	t := tritBot{
		c:       tritter.NewTritterClient(tCon),
		timeout: *sendTimeout,
		log:     log.NewLoggerClient(lCon),
	}

	for _, msg := range flag.Args() {
		if err := t.Send(context.Background(), log.InternalMessage{User: user.Username, Message: msg}); err != nil {
			glog.Fatalf("could not greet: %v", err)
		}
	}
	glog.Infof("Successfully sent %d messages", len(flag.Args()))
}

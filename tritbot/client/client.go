// A proxy that sends things on to tritter and logs the requests.
package main

import (
	"context"
	"errors"
	"flag"
	"os/user"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/mhutchinson/tritter/tritbot/log"
	"github.com/mhutchinson/tritter/tritter"
	"google.golang.org/grpc"
)

var (
	tritterAddr    = flag.String("tritter_addr", "localhost:50051", "the address of the tritter server")
	connectTimeout = flag.Duration("connect_timeout", time.Second, "the timeout for connecting to the server")
	sendTimeout    = flag.Duration("send_timeout", 5*time.Second, "the timeout for logging & sending each message")

	checkProof = flag.Bool("check_proof", false, "whether to confirm the data is logged before sending to tritter")
	loggerAddr = flag.String("logger_addr", "localhost:50052", "the address of the logger server")
)

type tritBot struct {
	c       tritter.TritterClient
	timeout time.Duration

	log log.LoggerClient
}

func (t *tritBot) Send(ctx context.Context, msg log.InternalMessage) error {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// First write the message to the log.
	r, err := t.log.Log(ctx, &log.LogRequest{Message: &msg})
	if err != nil {
		return err
	}

	// Second: check the message is in the log.
	if *checkProof {
		if r.GetProof() == nil {
			return errors.New("no proof to verify")
		}
		// TODO(mhutchinson): actually check the proof.
		glog.Warningf("TODO: check proof %v", r)
	}

	// Then continue to send the message to the server.
	_, err = t.c.Send(ctx, &tritter.SendRequest{Message: msg.GetMessage()})
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
		glog.Fatalf("did not connect to tritter on %v: %v", *tritterAddr, err)
	}
	defer tCon.Close()

	// Set up a connection to the Logger server.
	lCon, err := grpc.DialContext(ctx, *loggerAddr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		glog.Fatalf("did not connect to logger on %v: %v", *loggerAddr, err)
	}
	defer lCon.Close()

	t := tritBot{
		c:       tritter.NewTritterClient(tCon),
		timeout: *sendTimeout,
		log:     log.NewLoggerClient(lCon),
	}

	for _, msg := range flag.Args() {
		m := log.InternalMessage{
			User:      user.Username,
			Message:   msg,
			Timestamp: ptypes.TimestampNow(),
		}
		if err := t.Send(context.Background(), m); err != nil {
			glog.Fatalf("could not send message: %v", err)
		}
	}
	glog.Infof("Successfully sent %d messages", len(flag.Args()))
}

// A proxy that sends things on to tritter and logs the requests.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"
	pb "github.com/mhutchinson/tritter"
	"google.golang.org/grpc"
)

var (
	tritterAddr    = flag.String("tritter_addr", "localhost:50051", "the address of the tritter server")
	connectTimeout = flag.Duration("connect_timeout", time.Second, "the timeout for connecting to the server")
	sendTimeout    = flag.Duration("send_timeout", 100*time.Millisecond, "the timeout for sending each message")

	logFile = flag.String("log_file", "/tmp/tritter.log", "file path for message log")
)

type fileLogger struct {
	f *os.File
}

func (l *fileLogger) log(msg string) error {
	t := time.Now().UTC()
	_, err := l.f.WriteString(fmt.Sprintf("%v: %v\n", t.Format(time.RFC3339), msg))
	return err
}

type tritBot struct {
	c       pb.TritterClient
	timeout time.Duration
	log     fileLogger
}

func (t *tritBot) Send(ctx context.Context, msg string) error {
	glog.Infof("Sending message '%v'", msg)

	// First ensure the message is in the log.
	if err := t.log.log(msg); err != nil {
		glog.Fatalf("failed to log: %v", err)
	}

	// Then continue to send the message to the server.
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	_, err := t.c.Send(ctx, &pb.SendRequest{Message: msg})
	return err
}

func main() {
	flag.Parse()

	// Read the message from the argument list.
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

	// Open the log file for writing.
	f, err := os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		glog.Fatalf("could not open log file: %v", err)
	}

	t := tritBot{
		c:       pb.NewTritterClient(conn),
		timeout: *sendTimeout,
		log:     fileLogger{f: f},
	}

	for _, msg := range flag.Args() {
		if err := t.Send(context.Background(), msg); err != nil {
			glog.Fatalf("could not greet: %v", err)
		}
	}
	glog.Infof("Successfully sent %d messages", len(flag.Args()))
}

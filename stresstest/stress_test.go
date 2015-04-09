package stresstest

import (
	"io"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/docker/spdystream"
)

func TestStress(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				break
			}

			spdyConn, _ := spdystream.NewConnection(conn, true)
			go spdyConn.Serve(spdystream.NoOpStreamHandler)
		}
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Error dialing server: %s", err)
	}

	spdyConn, err := spdystream.NewConnection(conn, false)
	if err != nil {
		t.Fatalf("Error creating spdy connection: %s", err)
	}
	go spdyConn.Serve(spdystream.NoOpStreamHandler)

	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	streams := []*spdystream.Stream{}
	count := 1000
	for i := 1; i < count; i++ {
		stream, err := spdyConn.CreateStream(http.Header{}, nil, false)
		if err != nil {
			t.Fatalf("Error creating stream: %s", err)
		}

		if err := stream.Wait(); err != nil {
			t.Fatalf("Error waiting for stream: %s", err)
		}
		go io.Copy(devnull, stream)
		streams = append(streams, stream)
	}

	closer := make(chan struct{})
	data := []byte("x")
	go func() {
		for i, stream := range streams {
			go func() {
				for {
					stream.Write(data)
				}
			}()
			if i == count/2 {
				close(closer)
			}
		}
	}()

	<-closer
	if err := conn.Close(); err != nil {
		t.Fatalf("Error closing client connection: %s", err)
	}

	if err = listener.Close(); err != nil {
		t.Fatalf("Error closing listener: %s", err)
	}
}

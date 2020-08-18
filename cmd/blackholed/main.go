package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/jessevdk/go-flags"
)

const maxBufferSize = 4096

var opts struct {
	UDP  bool `short:"u" long:"udp" description:"Listen to UDP port instead of TCP port"`
	Port uint `short:"p" long:"port" default:"7468" description:"Port number to listen"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		panic(err)
	}

	lc := &net.ListenConfig{
		KeepAlive: 30 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go listenTCP(ctx, lc)

	intr := make(chan os.Signal, 1)
	signal.Notify(intr, os.Interrupt)
	<-intr
	close(intr)
	cancel()
	log.Println("Shutdown.")
}

func listenTCP(ctx context.Context, lc *net.ListenConfig) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", opts.Port))
	log.Printf("Listening to port:%d.\n", opts.Port)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Accept connection from ", conn.RemoteAddr())
		go io.Copy(ioutil.Discard, conn)
	}
}

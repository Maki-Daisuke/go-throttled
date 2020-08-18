package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"github.com/efarrer/iothrottler"
	"github.com/jessevdk/go-flags"
)

const maxBufferSize = 4096

var opts struct {
	UDP   bool `short:"u" long:"udp" description:"Use UDP instead of TCP"`
	Limit uint `short:"l" long:"limit" default:"1024" description:"Limit bandwidth by Kbps"`
}

func main() {
	args, err := flags.Parse(&opts)
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			os.Exit(0)
		}
		log.Fatal(err)
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s HOST:PORT\n", os.Args[0])
		os.Exit(1)
	}

	proto := "tcp"
	if opts.UDP {
		proto = "udp"
	}

	conn, err := net.Dial(proto, args[0])
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to ", conn.RemoteAddr())

	throttler := iothrottler.NewIOThrottlerPool(iothrottler.Kbps * iothrottler.Bandwidth(opts.Limit))
	defer throttler.ReleasePool()

	conn, err = throttler.AddConn(conn)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Throttling to %d Kbps\n", opts.Limit)

	// Discard all incoming data
	go io.Copy(ioutil.Discard, conn)

	arr := [maxBufferSize]byte{}
	buf := arr[:]
	var totalBytes, instantBytes uint64
	start := time.Now()
	prev := start
	for {
		rand.Read(buf) // Generate random data
		n, _ := conn.Write(buf)
		totalBytes += uint64(n)
		instantBytes += uint64(n)

		now := time.Now()
		if now.Sub(prev) >= 1*time.Second {
			bitrate := float64(instantBytes) * 8 / now.Sub(prev).Seconds()
			avBitrate := float64(totalBytes) * 8 / now.Sub(start).Seconds()
			fmt.Fprintf(os.Stderr, "\rBitrate: %s, Total sent: %s, Av. bitrate: %s", formatBitrate(bitrate), formatBytes(totalBytes), formatBitrate(avBitrate))
		}
	}
}

func formatBytes(n uint64) string {
	if n < 2*1024 {
		return fmt.Sprintf(`%d bytes`, n)
	}

	x := float64(n) / 1024
	if x < 2*1024 {
		return fmt.Sprintf(`%.2f KiB`, x)
	}

	x = x / 1024
	if x < 2*1024 {
		return fmt.Sprintf(`%.2f MiB`, x)
	}

	x = x / 1024
	return fmt.Sprintf(`%.2f GiB`, x)
}

func formatBitrate(x float64) string {
	if x < 2*1000 {
		return fmt.Sprintf(`%.2f bps`, x)
	}

	x = x / 1000
	if x < 2*1000 {
		return fmt.Sprintf(`%.2f Kbps`, x)
	}

	x = x / 1000
	if x < 2*1000 {
		return fmt.Sprintf(`%.2f Mbps`, x)
	}

	x = x / 1000
	return fmt.Sprintf(`%.2f Gbps`, x)
}

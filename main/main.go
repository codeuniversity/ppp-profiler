package main

import (
	"flag"

	profiler "github.com/codeuniversity/ppp-profiler"
)

func main() {
	var mhistHTTPAddress string
	var mhistTCPAddress string
	var port int
	flag.IntVar(&port, "port", 4000, "defines the port on which the profiler listens")
	flag.StringVar(&mhistHTTPAddress, "mhist_http_address", "http://localhost:6666", "The address to the mhist http endpoint")
	flag.StringVar(&mhistTCPAddress, "mhist_tcp_address", "localhost:6667", "The address to the mhist tcp endpoint. Listens for realtime updates on this address.")

	flag.Parse()

	server := profiler.NewServer(port, mhistHTTPAddress, mhistTCPAddress)
	server.Run()
}

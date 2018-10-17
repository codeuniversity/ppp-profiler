package main

import profiler "github.com/codeuniversity/ppp-profiler"

func main() {
	server := profiler.NewServer("localhost:6667")
	server.Run()
}

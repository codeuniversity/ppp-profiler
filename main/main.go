package main

import profiler "github.com/codeuniversity/ppp-profiler"

func main() {
	server := profiler.NewServer("http://localhost:6666", "localhost:6667")
	server.Run()
}

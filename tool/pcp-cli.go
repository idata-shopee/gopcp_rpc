package main

import (
	"flag"
	"fmt"
	"github.com/lock-free/gopcp"
	rpc "github.com/lock-free/gopcp_rpc"
	"github.com/lock-free/gopcp_stream"
	"time"
)

func main() {
	host := flag.String("host", "127.0.0.1", "host for pcp server")
	port := flag.Int("port", 4231, "port for pcp server")
	timeout := flag.Int("timeout", 300, "timeout for request")
	text := flag.String("code", "[\"List\", \"hello\"]", "code")

	flag.Parse()

	// create client
	if client, err := rpc.GetPCPRPCClient(*host, *port, func(*gopcp_stream.StreamServer) *gopcp.Sandbox {
		return gopcp.GetSandbox(map[string]*gopcp.BoxFunc{})
	}, func(e error) {
		if e != nil {
			panic(e)
		}
	}); err != nil {
		panic(err)
	} else {
		if ret, err := client.CallRemote(*text, time.Duration(*timeout)*time.Second); err != nil {
			panic(err)
		} else {
			fmt.Printf("%v", ret)
			client.Close()
		}
	}
}

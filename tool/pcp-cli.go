package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/lock-free/gopcp"
	rpc "github.com/lock-free/gopcp_rpc"
	"github.com/lock-free/gopcp_stream"
)

func main() {
	host := flag.String("host", "127.0.0.1", "host for pcp server")
	port := flag.Int("port", 4231, "port for pcp server")
	timeout := flag.Int("timeout", 300, "timeout for request")
	text := flag.String("code", "[\"List\", \"hello\"]", "code")

	flag.Parse()

	// create client
	client, err := rpc.GetPCPRPCClient(*host, *port, func(*gopcp_stream.StreamServer) *gopcp.Sandbox {
		return gopcp.GetSandbox(map[string]*gopcp.BoxFunc{})
	}, func(e error) {
		if e != nil {
			panic(e)
		}
	})

	if err != nil {
		panic(err)
	}
	defer client.Close()

	ret, err := client.CallRemote(*text, time.Duration(*timeout)*time.Second)

	if err != nil {
		panic(err)
	}

	bs, err := json.Marshal(ret)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s", string(bs))
}

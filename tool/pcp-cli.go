package main

import (
	"flag"
	"fmt"
	rpc "github.com/idata-shopee/gopcp_rpc"
	"time"
)

func main() {
	host := flag.String("host", "127.0.0.1", "host for pcp server")
	port := flag.Int("port", 4231, "port for pcp server")
	timeout := flag.Int("timeout", 300, "timeout for request")
	text := flag.String("code", "[\"List\", \"hello\"]", "code")

	flag.Parse()

	// create client
	if client, err := rpc.GetPCPRPCClient(*host, *port, func(e error) {
		panic(e)
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

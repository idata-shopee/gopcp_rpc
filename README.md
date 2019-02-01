# gopcp_rpc

A golang RPC based on pcp protocol

## Quick example

- server

```go
import (
  "errors"
	"github.com/idata-shopee/gopcp"
	rpc "github.com/idata-shopee/gopcp_rpc"
)

funcMap := map[string]*gopcp.BoxFunc{
	"add": gopcp.ToSandboxFun(func(args []interface{}, pcpServer *gopcp.PcpServer) (interface{}, error) {
		var res float64
		for _, arg := range args {
			if val, ok := arg.(float64); !ok {
				return nil, errors.New("args should be int")
			} else {
				res += val
			}
		}
		return res, nil
	}),
	"testError": gopcp.ToSandboxFun(func(args []interface{}, pcpServer *gopcp.PcpServer) (interface{}, error) {
		return nil, errors.New("errrrorrr")
	}),
}

sandBox := gopcp.GetSandbox(funcMap)

server, err := rpc.GetPCPRPCServer(0, sandbox)
if err != nil {
  panic(err)
}
```

- client

```go
import (
  "time"
	"github.com/idata-shopee/gopcp"
	rpc "github.com/idata-shopee/gopcp_rpc"
)

// create client
client, cerr := rpc.GetPCPRPCClient("127.0.0.1", 8081, func(e error) {})

if cerr != nil {
  panic(cerr)
}

p := gopcp.PcpClient{}
ret, rerr := client.Call(p.Call("add", 1, 2), 1000*time.Millisecond)
```

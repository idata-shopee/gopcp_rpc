package gopcp_rpc

import (
	"errors"
	"github.com/idata-shopee/gopcp"
	"testing"
	"time"
)

func simpleSandbox() *gopcp.Sandbox {
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

		"sum": gopcp.ToSandboxFun(func(args []interface{}, pcpServer *gopcp.PcpServer) (interface{}, error) {
			list, ok := args[0].([]interface{})
			if !ok {
				return nil, errors.New("args should be int list")
			}
			v := 0.0
			for _, item := range list {
				itemValue, iok := item.(float64)
				if !iok {
					return nil, errors.New("args should be int list")
				}
				v += itemValue
			}
			return v, nil
		}),
	}
	sandBox := gopcp.GetSandbox(funcMap)
	return sandBox
}

func testPCPRCPCallServer(t *testing.T, callResult gopcp.CallResult, expect interface{}) {
	server, err := GetPCPRPCServer(0, simpleSandbox())
	if err != nil {
		t.Errorf("fail to start server, %v", err)
	}
	defer server.Close()

	for i := 1; i < 1000; i++ {
		// create client
		client, cerr := GetPCPRPCClient("127.0.0.1", server.GetPort(), func(e error) {})

		if cerr != nil {
			t.Errorf("fail to start client, %v", cerr)
		}
		defer client.Close()

		ret, rerr := client.Call(callResult, 1000*time.Millisecond)
		if rerr != nil {
			t.Errorf("call errored, %v", rerr)
		}

		assertEqual(t, ret, expect, "")
	}
}

func TestPCPRPCBase(t *testing.T) {
	p := gopcp.PcpClient{}
	testPCPRCPCallServer(t, p.Call("add", 1, 2), 3.0)
	testPCPRCPCallServer(t, p.Call("add", 1, 2, 3), 6.0)
}

func TestPCPRPCBase2(t *testing.T) {
	p := gopcp.PcpClient{}
	testPCPRCPCallServer(t, p.Call("sum", []int{1, 2, 3, 4, 5}), 15.0)
}

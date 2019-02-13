package gopcp_rpc

import (
	"errors"
	"github.com/idata-shopee/gopcp"
	"sync"
	"testing"
	"time"
)

func simpleSandbox() *gopcp.Sandbox {
	funcMap := map[string]*gopcp.BoxFunc{
		"add": gopcp.ToSandboxFun(func(args []interface{}, attachment interface{}, pcpServer *gopcp.PcpServer) (interface{}, error) {
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

		"sum": gopcp.ToSandboxFun(func(args []interface{}, attachment interface{}, pcpServer *gopcp.PcpServer) (interface{}, error) {
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

		"testSleep": gopcp.ToSandboxFun(func(args []interface{}, attachment interface{}, pcpServer *gopcp.PcpServer) (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return 1, nil
		}),

		"testError": gopcp.ToSandboxFun(func(args []interface{}, attachment interface{}, pcpServer *gopcp.PcpServer) (interface{}, error) {
			return nil, errors.New("errrrorrr")
		}),
	}
	sandBox := gopcp.GetSandbox(funcMap)
	return sandBox
}

func testPCPRPCCallServer(expectFail bool, t *testing.T, callResult gopcp.CallResult, expect interface{}, timeout time.Duration) {
	server, err := GetPCPRPCServer(0, simpleSandbox())
	if err != nil {
		t.Errorf("fail to start server, %v", err)
	}
	defer server.Close()

	// create client
	client, cerr := GetPCPRPCClient("127.0.0.1", server.GetPort(), func(e error) {})

	if cerr != nil {
		t.Errorf("fail to start client, %v", cerr)
	}

	defer client.Close()

	// how many requests
	count := 1000

	var wg sync.WaitGroup
	wg.Add(count)

	runClient := func() {
		defer wg.Done()
		ret, rerr := client.Call(callResult, timeout)
		if rerr != nil && !expectFail {
			t.Errorf("call errored, %v", rerr)
		} else {
			assertEqual(t, ret, expect, "")
		}
	}

	for i := 0; i < count; i++ {
		go runClient()
	}

	wg.Wait()
}

func testPCPRPCPool(expectFail bool, t *testing.T, callResult gopcp.CallResult, expect interface{}, timeout time.Duration) {
	server, err := GetPCPRPCServer(0, simpleSandbox())
	if err != nil {
		t.Errorf("fail to start server, %v", err)
	}
	defer server.Close()

	pool := GetPCPRPCPool("127.0.0.1", server.GetPort(), 8, 3000*time.Millisecond)

	count := 1000

	var wg sync.WaitGroup
	wg.Add(count)

	runClient := func() {
		defer wg.Done()
		// create client
		item, cerr := pool.Get()
		if cerr != nil {
			t.Errorf("fail to get item from pool, %v", cerr)
		}
		client, ok := item.(*PCPConnectionHandler)
		if !ok {
			t.Errorf("can not convert client")
		}

		if cerr != nil {
			t.Errorf("fail to start client, %v", cerr)
		}

		ret, rerr := client.Call(callResult, timeout)
		if rerr != nil && !expectFail {
			t.Errorf("call errored, %v", rerr)
		} else {
			assertEqual(t, ret, expect, "")
		}
	}

	for i := 0; i < count; i++ {
		go runClient()
	}

	wg.Wait()
}

func testRPC(expectFail bool, t *testing.T, callResult gopcp.CallResult, expect interface{}) {
	testPCPRPCCallServer(expectFail, t, callResult, expect, 15000*time.Millisecond)
	testPCPRPCPool(expectFail, t, callResult, expect, 15000*time.Millisecond)
}

func TestPCPRPCBase(t *testing.T) {
	p := gopcp.PcpClient{}
	testRPC(false, t, p.Call("add", 1, 2), 3.0)
	testRPC(false, t, p.Call("add", 1, 2, 3), 6.0)
}

func TestPCPRPCBase2(t *testing.T) {
	p := gopcp.PcpClient{}
	testRPC(false, t, p.Call("sum", []int{1, 2, 3, 4, 5}), 15.0)
}

func TestPCPRPCSleep(t *testing.T) {
	p := gopcp.PcpClient{}
	testRPC(false, t, p.Call("testSleep"), 1.0)
}

func TestPCPRPCError(t *testing.T) {
	p := gopcp.PcpClient{}
	testRPC(true, t, p.Call("testError"), nil)
}

func TestPCPRPCMissingFun(t *testing.T) {
	p := gopcp.PcpClient{}
	testRPC(true, t, p.Call("fakkkkkkkkkk"), nil)
}

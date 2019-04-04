package gopcp_rpc

import (
	"errors"
	"github.com/idata-shopee/gopcp"
	"github.com/idata-shopee/gopcp_stream"
	"sync"
	"testing"
	"time"
)

func simpleSandbox(streamServer *gopcp_stream.StreamServer) *gopcp.Sandbox {
	funcMap := map[string]*gopcp.BoxFunc{
		"identity": gopcp.ToSandboxFun(func(args []interface{}, attachment interface{}, pcpServer *gopcp.PcpServer) (interface{}, error) {
			return args[0], nil
		}),

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
			time.Sleep(5 * time.Millisecond)
			return 1, nil
		}),

		"testError": gopcp.ToSandboxFun(func(args []interface{}, attachment interface{}, pcpServer *gopcp.PcpServer) (interface{}, error) {
			return nil, errors.New("errrrorrr")
		}),
	}
	sandBox := gopcp.GetSandbox(funcMap)
	return sandBox
}

// count: how many requests
func testPCPRPCCallServer(expectFail bool, t *testing.T, callResult gopcp.CallResult, expect interface{}, timeout time.Duration, count int) {
	server, err := GetPCPRPCServer(0, simpleSandbox)
	if err != nil {
		t.Errorf("fail to start server, %v", err)
	}
	defer server.Close()

	// create client
	client, cerr := GetPCPRPCClient("127.0.0.1", server.GetPort(), func(streamServer *gopcp_stream.StreamServer) *gopcp.Sandbox {
		return gopcp.GetSandbox(map[string]*gopcp.BoxFunc{})
	}, func(e error) {})

	if cerr != nil {
		t.Errorf("fail to start client, %v", cerr)
	}

	defer client.Close()

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

func testPCPRPCPool(expectFail bool, t *testing.T, callResult gopcp.CallResult, expect interface{}, timeout time.Duration, count int) {
	server, err := GetPCPRPCServer(0, simpleSandbox)
	if err != nil {
		t.Errorf("fail to start server, %v", err)
	}

	pool := GetPCPRPCPool(func() (string, int, error) {
		return "127.0.0.1", server.GetPort(), nil
	}, func(streamServer *gopcp_stream.StreamServer) *gopcp.Sandbox {
		return gopcp.GetSandbox(map[string]*gopcp.BoxFunc{})
	}, 8, 30*time.Millisecond, 30*time.Millisecond)
	defer pool.Shutdown()

	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(count)

	runClient := func() {
		defer wg.Done()
		// pickup a client
		if item, err := pool.Get(); err != nil {
			t.Errorf("fail to get item from pool, %v", err)
		} else if client, ok := item.(*PCPConnectionHandler); !ok {
			t.Errorf("can not convert client")
		} else if ret, err := client.Call(callResult, timeout); err != nil && !expectFail {
			t.Errorf("call errored, %v", err)
		} else {
			assertEqual(t, ret, expect, "")
		}
	}

	for i := 0; i < count; i++ {
		go runClient()
	}

	wg.Wait()
}

func testRPC(expectFail bool, t *testing.T, callResult gopcp.CallResult, expect interface{}, c1 int, c2 int) {
	testPCPRPCCallServer(expectFail, t, callResult, expect, 1000*time.Millisecond, c1)
	testPCPRPCPool(expectFail, t, callResult, expect, 15000*time.Millisecond, c2)
}

func TestPCPRPCBase(t *testing.T) {
	p := gopcp.PcpClient{}
	testRPC(false, t, p.Call("add", 1, 2), 3.0, 500, 500)
	testRPC(false, t, p.Call("add", 1, 2, 3), 6.0, 500, 500)
}

func TestPCPRPCBase2(t *testing.T) {
	p := gopcp.PcpClient{}
	testRPC(false, t, p.Call("sum", []int{1, 2, 3, 4, 5}), 15.0, 500, 500)
}

func TestPCPRPCSleep(t *testing.T) {
	p := gopcp.PcpClient{}
	testRPC(false, t, p.Call("testSleep"), 1.0, 500, 500)
}

func TestPCPRPCError(t *testing.T) {
	p := gopcp.PcpClient{}
	testRPC(true, t, p.Call("testError"), nil, 100, 100)
}

func TestPCPRPCMissingFun(t *testing.T) {
	p := gopcp.PcpClient{}
	testRPC(true, t, p.Call("fakkkkkkkkkk"), nil, 100, 100)
}

func TestStream(t *testing.T) {
	server, err := GetPCPRPCServer(0, func(streamServer *gopcp_stream.StreamServer) *gopcp.Sandbox {
		return gopcp.GetSandbox(map[string]*gopcp.BoxFunc{
			// define a stream pai
			"streamApi": streamServer.StreamApi(func(
				streamProducer gopcp_stream.StreamProducer,
				args []interface{},
				attachment interface{},
				pcpServer *gopcp.PcpServer,
			) (interface{}, error) {
				seed := args[0].(string)
				streamProducer.SendData(seed+"1", 10*time.Second)
				streamProducer.SendData(seed+"2", 10*time.Second)
				streamProducer.SendData(seed+"3", 10*time.Second)
				streamProducer.SendEnd(10 * time.Second)
				return nil, nil
			}),
		})
	})
	if err != nil {
		t.Errorf("fail to start server, %v", err)
	}
	defer server.Close()

	client, err := GetPCPRPCClient("127.0.0.1", server.GetPort(), func(streamServer *gopcp_stream.StreamServer) *gopcp.Sandbox {
		return gopcp.GetSandbox(map[string]*gopcp.BoxFunc{})
	}, func(e error) {})
	if err != nil {
		t.Errorf("fail to connect, %v", err)
	}

	for i := 1; i < 100; i++ {
		ret := ""
		// call stream api
		exp, _ := client.streamClient.StreamCall("streamApi", "_", func(t int, d interface{}) {
			if t == gopcp_stream.STREAM_DATA {
				ret += d.(string)
			}
		})
		client.Call(*exp, 10*time.Second)
		assertEqual(t, ret, "_1_2_3", "")
	}
}

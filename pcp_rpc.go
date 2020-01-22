package gopcp_rpc

import (
	"github.com/lock-free/goaio"
	"github.com/lock-free/gopcp"
	"github.com/lock-free/gopcp_stream"
	"github.com/lock-free/gopool"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

// generic connection interface for tcp client and server both
type GetTcpConn = func(goaio.BytesReadHandler, goaio.OnCloseHandler) (goaio.ConnectionHandler, error)

type GenerateSandbox = func(*gopcp_stream.StreamServer) *gopcp.Sandbox

func GetPcpConnectionHandlerFromTcpConn(t int, generateSandbox GenerateSandbox, getTcpConn GetTcpConn) (*PCPConnectionHandler, error) {
	var pcpConnectionHandler *PCPConnectionHandler
	var remoteCallMap sync.Map

	pcpClient := gopcp.PcpClient{}

	// create stream object
	streamClient := gopcp_stream.GetStreamClient()
	streamServer := gopcp_stream.GetStreamServer(STREAM_ACCEPT_NAME, func(command string, timeout time.Duration) (interface{}, error) {
		return pcpConnectionHandler.CallRemote(command, timeout)
	})
	// default stream accept api
	boxMap := map[string]*gopcp.BoxFunc{}
	boxMap[STREAM_ACCEPT_NAME] = gopcp_stream.GetPcpStreamAcceptBoxFun(streamClient)

	// create pcp server
	pcpServer := gopcp.NewPcpServer(gopcp.GetSandbox(boxMap).Extend(generateSandbox(streamServer)))

	pcpConnectionHandler = &PCPConnectionHandler{packageProtocol: GetPackageProtocol(),
		PcpClient:     pcpClient,
		pcpServer:     pcpServer,
		ConnHandler:   nil,
		remoteCallMap: remoteCallMap,
		StreamClient:  streamClient,
	}

	if connHandler, err := getTcpConn(pcpConnectionHandler.OnData, func(error) {
		pcpConnectionHandler.Clean()
	}); err != nil {
		return nil, err
	} else {
		pcpConnectionHandler.ConnHandler = &connHandler
		if t == 1 {
			go connHandler.ReadFromConn()
		}
		return pcpConnectionHandler, nil
	}
}

// connection onConnected, onClose
type ConnectionEvent struct {
	OnClose     goaio.OnCloseHandler
	OnConnected OnConnectedHandler
}

type OnConnectedHandler = func(*PCPConnectionHandler)

// build pcp rpc server based on the tcp server itself
func GetPCPRPCServer(port int, generateSandbox GenerateSandbox, cer func() *ConnectionEvent) (*goaio.TcpServer, error) {
	if tcpServer, err := goaio.GetTcpServer(port, func(conn net.Conn) goaio.ConnectionHandler {
		var connHandler goaio.ConnectionHandler
		var ce *ConnectionEvent = nil
		if cer != nil {
			ce = cer()
		}

		pcpConnectionHandler, _ := GetPcpConnectionHandlerFromTcpConn(0, generateSandbox, func(onData goaio.BytesReadHandler, onClose goaio.OnCloseHandler) (goaio.ConnectionHandler, error) {
			connHandler = goaio.GetConnectionHandler(conn, onData, func(err error) {
				if ce != nil {
					ce.OnClose(err)
				}
				onClose(err)
			})
			return connHandler, nil
		})

		if ce != nil {
			go ce.OnConnected(pcpConnectionHandler)
		}

		return connHandler
	}); err != nil {
		return nil, err
	} else {
		go tcpServer.Accepts()
		return tcpServer, err
	}
}

// build pcp client based on the tcp client itself
func GetPCPRPCClient(host string, port int, generateSandbox GenerateSandbox, onClose goaio.OnCloseHandler) (*PCPConnectionHandler, error) {
	return GetPcpConnectionHandlerFromTcpConn(1, generateSandbox, func(onData goaio.BytesReadHandler, closeHandle goaio.OnCloseHandler) (goaio.ConnectionHandler, error) {
		return goaio.GetTcpClient(host, port, onData, func(err error) {
			closeHandle(err)
			if onClose != nil {
				onClose(err)
			}
		})
	})
}

// return host and port
type GetAddress = func() (string, int, error)

// build pcp pool based on the tcp client
func GetPCPRPCPool(getAddress GetAddress, generateSandbox GenerateSandbox, poolSize int, duration time.Duration, retryDuration time.Duration) *gopool.Pool {
	getNewItem := func(onItemBoken gopool.OnItemBorken) (*gopool.Item, error) {
		if host, port, err := getAddress(); err != nil {
			return nil, err
		} else {
			if pcpConnectionHandler, err := GetPcpConnectionHandlerFromTcpConn(1, generateSandbox, func(onData goaio.BytesReadHandler, closeHandle goaio.OnCloseHandler) (goaio.ConnectionHandler, error) {
				return goaio.GetTcpClient(host, port, onData, func(err error) {
					log.Printf("connection closed! remote-host=%s, remote-port=%s, errMsg=%s\n", host, strconv.Itoa(port), err)
					closeHandle(err)
					onItemBoken()
				})
			}); err != nil {
				log.Printf("connect failed! host=%s, port=%s, errMsg=%s\n", host, strconv.Itoa(port), err)
				return nil, err
			} else {
				log.Printf("connected host=%s, port=%s\n", host, strconv.Itoa(port))
				return &gopool.Item{pcpConnectionHandler, func() {
					pcpConnectionHandler.Close()
				}}, nil
			}
		}
	}

	return gopool.GetPool(getNewItem, poolSize, duration, retryDuration)
}

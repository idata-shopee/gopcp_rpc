package gopcp_rpc

import (
	"github.com/idata-shopee/goaio"
	"github.com/idata-shopee/gopcp"
	"github.com/idata-shopee/gopcp_stream"
	"github.com/idata-shopee/gopool"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

// generic connection interface for tcp client and server both
type GetTcpConn = func(goaio.BytesReadHandler, goaio.OnCloseHandler) (goaio.ConnectionHandler, error)

type GenerateSandbox = func(*gopcp_stream.StreamServer) *gopcp.Sandbox

func GetPcpConnectionHandlerFromTcpConn(generateSandbox GenerateSandbox, getTcpConn GetTcpConn) (*PCPConnectionHandler, error) {
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
		pcpClient:     pcpClient,
		pcpServer:     pcpServer,
		connHandler:   nil,
		remoteCallMap: remoteCallMap,
		streamClient:  streamClient,
	}

	if connHandler, err := getTcpConn(pcpConnectionHandler.OnData, func(error) {
		pcpConnectionHandler.Clean()
	}); err != nil {
		return nil, err
	} else {
		pcpConnectionHandler.connHandler = &connHandler
		return pcpConnectionHandler, nil
	}
}

// build pcp rpc server based on the tcp server itself
func GetPCPRPCServer(port int, generateSandbox GenerateSandbox) (*goaio.TcpServer, error) {
	if tcpServer, err := goaio.GetTcpServer(port, func(conn net.Conn) goaio.ConnectionHandler {
		var connHandler goaio.ConnectionHandler

		// TODO handle error when get pcp connection handler
		GetPcpConnectionHandlerFromTcpConn(generateSandbox, func(onData goaio.BytesReadHandler, onClose goaio.OnCloseHandler) (goaio.ConnectionHandler, error) {
			connHandler = goaio.GetConnectionHandler(conn, onData, func(err error) {
				onClose(err)
				// TODO, handle when connection closed
			})
			return connHandler, nil
		})

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
	return GetPcpConnectionHandlerFromTcpConn(generateSandbox, func(onData goaio.BytesReadHandler, closeHandle goaio.OnCloseHandler) (goaio.ConnectionHandler, error) {
		return goaio.GetTcpClient(host, port, onData, func(err error) {
			closeHandle(err)
			onClose(err)
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
			if pcpConnectionHandler, err := GetPcpConnectionHandlerFromTcpConn(generateSandbox, func(onData goaio.BytesReadHandler, closeHandle goaio.OnCloseHandler) (goaio.ConnectionHandler, error) {
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

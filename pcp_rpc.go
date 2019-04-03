package gopcp_rpc

import (
	"github.com/idata-shopee/goaio"
	"github.com/idata-shopee/gopcp"
	"github.com/idata-shopee/gopool"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

// generic connection interface for tcp client and server both
type GetTcpConn = func(goaio.BytesReadHandler) (goaio.ConnectionHandler, error)

func GetPcpConnectionHandlerFromTcpConn(sandbox *gopcp.Sandbox, getTcpConn GetTcpConn) (*PCPConnectionHandler, error) {
	pcpClient := gopcp.PcpClient{}
	pcpServer := gopcp.NewPcpServer(sandbox)

	var remoteCallMap sync.Map
	pcpConnectionHandler := PCPConnectionHandler{GetPackageProtocol(), pcpClient, pcpServer, nil, remoteCallMap}
	if connHandler, err := getTcpConn(pcpConnectionHandler.OnData); err != nil {
		return nil, err
	} else {
		pcpConnectionHandler.connHandler = &connHandler
		return &pcpConnectionHandler, nil
	}

}

// build pcp rpc server based on the tcp server itself
func GetPCPRPCServer(port int, sandbox *gopcp.Sandbox) (*goaio.TcpServer, error) {
	if tcpServer, err := goaio.GetTcpServer(port, func(conn net.Conn) goaio.ConnectionHandler {
		var connHandler goaio.ConnectionHandler

		GetPcpConnectionHandlerFromTcpConn(sandbox, func(onData goaio.BytesReadHandler) (goaio.ConnectionHandler, error) {
			connHandler = goaio.GetConnectionHandler(conn, onData, func(err error) {
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
func GetPCPRPCClient(host string, port int, onClose goaio.OnCloseHandler) (*PCPConnectionHandler, error) {
	return GetPcpConnectionHandlerFromTcpConn(gopcp.GetSandbox(map[string]*gopcp.BoxFunc{}), func(onData goaio.BytesReadHandler) (goaio.ConnectionHandler, error) {
		return goaio.GetTcpClient(host, port, onData, onClose)
	})
}

// return host and port
type GetAddress = func() (string, int, error)

// build pcp pool based on the tcp client
func GetPCPRPCPool(getAddress GetAddress, poolSize int, duration time.Duration, retryDuration time.Duration) *gopool.Pool {
	getNewItem := func(onItemBoken gopool.OnItemBorken) (*gopool.Item, error) {
		if host, port, err := getAddress(); err != nil {
			return nil, err
		} else {
			pcpClient := gopcp.PcpClient{}
			pcpServer := gopcp.NewPcpServer(gopcp.GetSandbox(map[string]*gopcp.BoxFunc{}))
			var remoteCallMap sync.Map
			pcpConnectionHandler := PCPConnectionHandler{GetPackageProtocol(), pcpClient, pcpServer, nil, remoteCallMap}

			if tcpClient, err := goaio.GetTcpClient(host, port, pcpConnectionHandler.OnData, func(err error) {
				log.Printf("connection closed! remote-host=%s, remote-port=%s, errMsg=%s\n", host, strconv.Itoa(port), err)
				onItemBoken()
			}); err != nil {
				log.Printf("connect failed! host=%s, port=%s, errMsg=%s\n", host, strconv.Itoa(port), err)
				return nil, err
			} else {
				pcpConnectionHandler.connHandler = &tcpClient
				log.Printf("connected host=%s, port=%s\n", host, strconv.Itoa(port))
				return &gopool.Item{&pcpConnectionHandler, func() {
					pcpConnectionHandler.Close()
				}}, nil
			}
		}
	}

	return gopool.GetPool(getNewItem, poolSize, duration, retryDuration)
}

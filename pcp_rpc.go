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

type OnCloseHandler = func(error)

func GetPCPRPCServer(port int, sandbox *gopcp.Sandbox) (*goaio.TcpServer, error) {
	tcpServer, err := goaio.GetTcpServer(port, func(conn net.Conn) goaio.ConnectionHandler {
		pcpClient := gopcp.PcpClient{}
		pcpServer := gopcp.NewPcpServer(sandbox)
		var remoteCallMap sync.Map
		pcpConnectionHandler := PCPConnectionHandler{GetPackageProtocol(), pcpClient, pcpServer, nil, remoteCallMap}
		connHandler := goaio.GetConnectionHandler(conn, pcpConnectionHandler.OnData, func(err error) {
			// TODO, handle when connection closed
		})

		pcpConnectionHandler.connHandler = &connHandler
		return connHandler
	})

	if err == nil {
		go tcpServer.Accepts()
	}

	return tcpServer, err
}

func GetPCPRPCClient(host string, port int, onClose OnCloseHandler) (*PCPConnectionHandler, error) {
	pcpClient := gopcp.PcpClient{}
	pcpServer := gopcp.NewPcpServer(gopcp.GetSandbox(map[string]*gopcp.BoxFunc{}))
	var remoteCallMap sync.Map
	pcpConnectionHandler := PCPConnectionHandler{GetPackageProtocol(), pcpClient, pcpServer, nil, remoteCallMap}
	tcpClient, err := goaio.GetTcpClient(host, port, pcpConnectionHandler.OnData, onClose)

	pcpConnectionHandler.connHandler = &tcpClient

	if err != nil {
		return nil, err
	}

	return &pcpConnectionHandler, nil
}

// return host and port
type GetAddress = func() (string, int, error)

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

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

func GetPCPRPCServer(port int, sandbox *gopcp.Sandbox) (goaio.TcpServer, error) {
	tcpServer, err := goaio.GetTcpServer(port, func(conn net.Conn) goaio.ConnectionHandler {
		pcpClient := gopcp.PcpClient{}
		pcpServer := gopcp.NewPcpServer(sandbox)
		var remoteCallMap sync.Map
		pcpConnectionHandler := PCPConnectionHandler{GetPackageProtocol(), pcpClient, pcpServer, nil, remoteCallMap}
		connHandler := goaio.ConnectionHandler{conn, pcpConnectionHandler.OnData, func(err error) {
			// TODO, handle when connection closed
		}}

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

func GetPCPRPCPool(host string, port int, poolSize int, duration time.Duration) gopool.Pool {
	getNewItem := func(onItemBoken gopool.OnItemBorken) (*gopool.Item, error) {
		pcpClient := gopcp.PcpClient{}
		pcpServer := gopcp.NewPcpServer(gopcp.GetSandbox(map[string]*gopcp.BoxFunc{}))
		var remoteCallMap sync.Map
		pcpConnectionHandler := PCPConnectionHandler{GetPackageProtocol(), pcpClient, pcpServer, nil, remoteCallMap}
		tcpClient, err := goaio.GetTcpClient(host, port, pcpConnectionHandler.OnData, func(err error) {
			log.Println("connection closed! host=" + host + ", port=" + strconv.Itoa(port))
			onItemBoken()
		})

		pcpConnectionHandler.connHandler = &tcpClient

		if err != nil {
			log.Println("connect failed! host=" + host + ", port=" + strconv.Itoa(port))
			return nil, err
		}

		log.Println("connected, host=" + host + ", port=" + strconv.Itoa(port))
		return &gopool.Item{&pcpConnectionHandler, func() {
			pcpConnectionHandler.Close()
		}}, nil
	}

	return gopool.GetPool(getNewItem, poolSize, duration)
}

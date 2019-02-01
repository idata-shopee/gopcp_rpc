package gopcp_rpc

import (
	"encoding/binary"
	"github.com/idata-shopee/goaio"
	"sync"
)

// package header
// Bytes:  0      1     2    3    4
//      version [    body size      ]
// TODO checksum for safety

var headerLen = 5

func TextToPkt(text string) []byte {
	bytes := []byte(text)
	lenOfTextBytes := len(bytes)

	// body size bytes
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(lenOfTextBytes))
	return append(append([]byte{0}, lenBytes...), bytes...)
}

type PackageProtocol struct {
	buffer       []byte
	bufferLocker *sync.Mutex

	sentLock *sync.Mutex
}

// TODO sync?
func (p *PackageProtocol) SendPackage(connHandler *goaio.ConnectionHandler, text string) error {
	p.sentLock.Lock()
	defer p.sentLock.Unlock()
	return connHandler.SendBytes(TextToPkt(text))
}

func (p *PackageProtocol) GetPktText(data []byte) []string {
	p.bufferLocker.Lock()
	defer p.bufferLocker.Unlock()

	p.buffer = append(p.buffer, data...)

	var result []string
	for pktText := p.getSinglePkt(); pktText != ""; pktText = p.getSinglePkt() {
		result = append(result, pktText)
	}
	return result
}

func (p *PackageProtocol) Reset() {
	p.bufferLocker.Lock()
	defer p.bufferLocker.Unlock()

	var empty []byte
	p.buffer = empty
}

func (p *PackageProtocol) getSinglePkt() string {
	if len(p.buffer) <= headerLen {
		return ""
	}

	bodyLen := binary.BigEndian.Uint32(p.buffer[1:5])
	pktLen := headerLen + int(bodyLen)

	if len(p.buffer) >= pktLen {
		pkt := p.buffer[headerLen:pktLen]
		// update buffer
		p.buffer = p.buffer[pktLen:]
		return string(pkt)
	} else {
		return ""
	}
}

func GetPackageProtocol() *PackageProtocol {
	var bytes []byte
	var bufferLock = &sync.Mutex{}
	var sentLock = &sync.Mutex{}
	return &PackageProtocol{bytes, bufferLock, sentLock}
}

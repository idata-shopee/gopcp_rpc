package gopcp_rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/idata-shopee/goaio"
	"github.com/idata-shopee/gopcp"
	"github.com/satori/go.uuid"
	"strconv"
	"sync"
	"time"
)

type CommandData struct {
	Text   interface{} `json:"text"`
	Errno  int         `json:"errno"`
	ErrMsg string      `json:"errMsg"`
}

type CommandPkt struct {
	Id    string      `json:"id"`
	Ctype string      `json:"ctype"`
	Data  CommandData `json:"data"`
}

func stringToCommand(text string) (*CommandPkt, error) {
	var cmd CommandPkt
	err := json.Unmarshal([]byte(text), &cmd)
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

func commandToText(cmd CommandPkt) (string, error) {
	bytes, err := json.Marshal(cmd)
	if err != nil {
		return "", err
	}
	return string(bytes[:]), nil
}

var REQUEST_C_TYPE = "purecall-request"
var RESPONSE_C_TYPE = "purecall-response"

func getErrorMessage(err error) string {
	return err.Error()
}

func executeRequestCommand(requestCommand *CommandPkt, pcpServer *gopcp.PcpServer) (interface{}, error) {
	// request command
	text, ok := requestCommand.Data.Text.(string)
	if !ok {
		return nil, errors.New("Expect string for request command.")
	}

	return pcpServer.Execute(text, nil)
}

func packResponse(id string, text interface{}, err error) CommandPkt {
	var commandData *CommandData = nil
	if err != nil {
		commandData = &CommandData{text, 530, getErrorMessage(err)}
	} else {
		commandData = &CommandData{text, 0, ""}
	}
	return CommandPkt{id, RESPONSE_C_TYPE, *commandData}
}

type CallChannel struct {
	data interface{}
	err  error
}

type PCPConnectionHandler struct {
	packageProtocol *PackageProtocol
	pcpClient       gopcp.PcpClient
	pcpServer       *gopcp.PcpServer
	connHandler     *goaio.ConnectionHandler
	remoteCallMap   sync.Map
}

func (p *PCPConnectionHandler) OnData(chunk []byte) {
	texts := p.packageProtocol.GetPktText(chunk)
	go p.onDataHelp(texts) // execute may be slow, need to run at a seperated goroutine
}

func (p *PCPConnectionHandler) onDataHelp(texts []string) {
	for _, text := range texts {
		cmd, err := stringToCommand(text)
		if err != nil {
			// reset protocol
			p.packageProtocol.Reset()
			// can not trust rest data either
			break
		}

		switch ctype := cmd.Ctype; ctype {
		case REQUEST_C_TYPE:
			// handle request from client
			result, err := executeRequestCommand(cmd, p.pcpServer)
			cmdText, cerr := commandToText(packResponse(cmd.Id, result, err))
			if cerr != nil {
				fmt.Printf("fail to convert command to string: %v", cerr)
				fmt.Println()
			}

			sentErr := p.packageProtocol.SendPackage(p.connHandler, cmdText)
			if sentErr != nil {
				fmt.Printf("fail to sent package: %v", sentErr)
				fmt.Println()
			}

		case RESPONSE_C_TYPE:
			// handle response from server
			ch_raw, ok := p.remoteCallMap.Load(cmd.Id)
			if !ok {
				fmt.Printf("missing-pkt-id: can not find id %v in remote call map. Cmd content is %v. Normally, when timeout, the id also will be removed from remote call map.", cmd.Id, text)
				fmt.Println()
				return
			}
			// delete key
			p.remoteCallMap.Delete(cmd.Id)
			// pass to channel
			ch, _ := ch_raw.(chan CallChannel)
			if cmd.Data.Errno == 0 {
				ch <- CallChannel{cmd.Data.Text, nil}
			} else {
				errText := cmd.Data.ErrMsg + "(" + strconv.Itoa(cmd.Data.Errno) + ")"
				ch <- CallChannel{nil, errors.New(errText)}
			}
		default:
			// impossible
			fmt.Printf("unknown type of package. Type is %v", ctype)
		}
	}
}

func (p *PCPConnectionHandler) CallRemote(command string, timeout time.Duration) (interface{}, error) {
	// generate package with unique id
	uid, uerr := uuid.NewV4()
	if uerr != nil {
		return nil, uerr
	}
	id := uid.String()
	data := CommandPkt{id, REQUEST_C_TYPE, CommandData{command, 0, ""}}

	cmdText, cerr := commandToText(data)
	if cerr != nil {
		return nil, cerr
	}

	// register channel
	ch := make(chan CallChannel)
	p.remoteCallMap.Store(id, ch)

	// send package through connection
	serr := p.packageProtocol.SendPackage(p.connHandler, cmdText)
	if serr != nil {
		p.remoteCallMap.Delete(id)
		return nil, serr
	}

	// timeout action
	go p.timeoutChannel(id, ch, command, timeout)

	// wait for channel
	ret := <-ch

	if ret.err != nil {
		return nil, ret.err
	}
	return ret.data, nil
}

func (p *PCPConnectionHandler) timeoutChannel(id string, ch chan CallChannel, command string, timeout time.Duration) {
	time.Sleep(timeout)
	ch <- CallChannel{nil, errors.New("timeout for call. Command is " + command)}
	// p.remoteCallMap.Delete(id)
}

func (p *PCPConnectionHandler) Call(list gopcp.CallResult, timeout time.Duration) (interface{}, error) {
	cmdText, err := p.pcpClient.ToJSON(list)

	if err != nil {
		return nil, err
	}
	return p.CallRemote(cmdText, timeout)
}

func (p *PCPConnectionHandler) Close() {
	p.connHandler.Close(nil)
}

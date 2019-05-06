package gopcp_rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/idata-shopee/goaio"
	"github.com/idata-shopee/gopcp"
	"github.com/idata-shopee/gopcp_stream"
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

func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

func stringToCommand(text string) (*CommandPkt, error) {
	var cmd CommandPkt
	if err := json.Unmarshal([]byte(text), &cmd); err != nil {
		return nil, err
	} else {
		return &cmd, nil
	}
}

func commandToText(cmd CommandPkt) (string, error) {
	if bytes, err := JSONMarshal(cmd); err != nil {
		return "", err
	} else {
		return string(bytes[:]), nil
	}
}

var REQUEST_C_TYPE = "purecall-request"
var RESPONSE_C_TYPE = "purecall-response"

func getErrorMessage(err error) string {
	return err.Error()
}

func executeRequestCommand(requestCommand *CommandPkt, pcpServer *gopcp.PcpServer, pch *PCPConnectionHandler) (interface{}, error) {
	// request command
	if text, ok := requestCommand.Data.Text.(string); !ok {
		return nil, errors.New("Expect string for request command.")
	} else {
		// add pch as default attribute to attachment of pcp execution
		return pcpServer.Execute(text, map[string]interface{}{
			"pch": pch,
		})
	}
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
	PcpClient       gopcp.PcpClient
	pcpServer       *gopcp.PcpServer
	connHandler     *goaio.ConnectionHandler
	remoteCallMap   sync.Map
	StreamClient    *gopcp_stream.StreamClient
}

func (p *PCPConnectionHandler) OnData(chunk []byte) {
	texts := p.packageProtocol.GetPktText(chunk)
	go p.onDataHelp(texts) // execute may be slow, need to run at a seperated goroutine
}

func (p *PCPConnectionHandler) onDataHelp(texts []string) {
	for _, text := range texts {
		if cmd, err := stringToCommand(text); err != nil {
			// reset protocol
			// can not trust rest data either
			p.packageProtocol.Reset()
			break
		} else {
			switch ctype := cmd.Ctype; ctype {
			case REQUEST_C_TYPE:
				// handle request from client
				result, err := executeRequestCommand(cmd, p.pcpServer, p)

				if cmdText, err := commandToText(packResponse(cmd.Id, result, err)); err != nil {
					// TODO do more than just log
					fmt.Printf("fail to convert command to string: %v\n", err)
				} else if err = p.packageProtocol.SendPackage(p.connHandler, cmdText); err != nil {
					fmt.Printf("fail to sent package: %v\n", err)
				}

			case RESPONSE_C_TYPE:
				// handle response from server
				if ch_raw, ok := p.remoteCallMap.Load(cmd.Id); !ok {
					fmt.Printf("missing-pkt-id: can not find id %v in remote call map. Cmd content is %v. Normally, when timeout, the id also will be removed from remote call map.\n", cmd.Id, text)
				} else {
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
				}

			default:
				// impossible
				fmt.Printf("unknown type of package. Type is %v\n", ctype)
			}
		}
	}
}

func (p *PCPConnectionHandler) CallRemote(command string, timeout time.Duration) (interface{}, error) {
	// generate package with unique id
	uid := uuid.NewV4()

	id := uid.String()
	data := CommandPkt{id, REQUEST_C_TYPE, CommandData{command, 0, ""}}

	if cmdText, err := commandToText(data); err != nil {
		return nil, err
	} else {
		// register channel
		ch := make(chan CallChannel)
		p.remoteCallMap.Store(id, ch)

		// send package through connection
		if err := p.packageProtocol.SendPackage(p.connHandler, cmdText); err != nil {
			p.remoteCallMap.Delete(id)
			return nil, err
		} else {
			// timeout action
			go p.timeoutChannel(id, ch, command, timeout)

			// wait for channel
			ret := <-ch

			if ret.err != nil {
				return nil, ret.err
			} else {
				return ret.data, nil
			}
		}
	}
}

func (p *PCPConnectionHandler) timeoutChannel(id string, ch chan CallChannel, command string, timeout time.Duration) {
	time.Sleep(timeout)
	ch <- CallChannel{nil, errors.New("timeout for call. Command is " + command + " timeout=" + timeout.String())}
	p.remoteCallMap.Delete(id)
}

func (p *PCPConnectionHandler) Call(list gopcp.CallResult, timeout time.Duration) (interface{}, error) {
	if cmdText, err := p.PcpClient.ToJSON(list); err != nil {
		return nil, err
	} else {
		return p.CallRemote(cmdText, timeout)
	}
}

func (p *PCPConnectionHandler) Close() {
	p.connHandler.Close(nil)
	p.Clean()
}

func (p *PCPConnectionHandler) Clean() {
	p.StreamClient.Clean()
}

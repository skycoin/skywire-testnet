package terminalmanager

import (
	"net/rpc"
)

type RPCClient struct {
	Client *rpc.Client
}

type RPCMessage struct {
	Command   string
	Arguments []string
}

func RunClient(addr string) (*RPCClient, error) {
	rpcClient := &RPCClient{}
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		return nil, err
	}
	rpcClient.Client = client
	return rpcClient, nil
}

func (rpcClient *RPCClient) SendToRPC(command string, args []string) ([]byte, error) {
	msg := RPCMessage{
		Command:   command,
		Arguments: args}
	var result []byte
	err := rpcClient.Client.Call("RPCReceiver."+msg.Command, msg.Arguments,
		&result)
	return result, err
}

func (rpcClient *RPCClient) ErrorOut(err error) {
	println("Error. Server says:\n", err.Error())
}

package main

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/code-cord/cc.core.server/server"
	"github.com/sirupsen/logrus"
)

/*
import (
	"bufio"
	"log"
	"net/rpc/jsonrpc"
	"os"
)

type Reply struct {
	Data string
}

func main() {
	client, err := jsonrpc.Dial("tcp", "localhost:12345")
	if err != nil {
		log.Fatal(err)
	}
	in := bufio.NewReader(os.Stdin)
	for {
		line, _, err := in.ReadLine()
		if err != nil {
			log.Fatal(err)
		}
		var reply Reply
		err = client.Call("Listener.GetLine", line, &reply)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Reply: %v, Data: %v", reply, reply.Data)
	}
}
*/

//go:embed server.json
var serverInfo []byte

func main() {
	s, err := newServer()
	if err != nil {
		logrus.Fatalf("could not create server instance: %v", err)
	}

	s.NewStream()

	s.Run(context.Background())
}

func newServer() (*server.Server, error) {
	var info map[string]interface{}
	if err := json.Unmarshal(serverInfo, &info); err != nil {
		return nil, err
	}

	s := server.New(
		server.Address("127.0.0.1:8989"), // TODO: from cli
		server.Name(info["name"].(string)),
		server.Description(info["description"].(string)),
		server.Version(info["version"].(string)),
		server.Meta(info["meta"].(map[string]interface{})),
	)

	return &s, nil
}

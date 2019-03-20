package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

/* Runs HTTP API tests of chat application between 2 skywire-node`s

Set envvar SKYWIRE_INTEGRATION_TESTS=1 to enable them
Set SKYWIRE_HOST to the first skywire-node's address
Set SKYWIRE_NODE to the second skywire-node's address
Set SKYWIRE_NODE_PK to static_public_key of SKYWIRE_NODE

E.g.
```bash
export SKYWIRE_INTEGRATION_TESTS=1
export SKYWIRE_HOST=http://localhost:8000
export SKYWIRE_NODE=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' SKY01)
export SKYWIRE_NODE_PK=$(<./node/PK)
export SKYWIRE_HOST_PK=$(<./PK)
```
*/

// Test suite for testing chat between 2 nodes
type TwoNodesSuite struct {
	suite.Suite
	Disabled bool
	Host     string
	HostPK   string
	Node     string
	NodePK   string
}

func (suite *TwoNodesSuite) SetupTest() {
	envEnabled := os.Getenv("SKYWIRE_INTEGRATION_TESTS")
	suite.Disabled = (envEnabled != "1")
	suite.Host = os.Getenv("SKYWIRE_HOST")
	suite.HostPK = os.Getenv("SKYWIRE_HOST_PK")
	suite.Node = os.Getenv("SKYWIRE_NODE")
	suite.NodePK = os.Getenv("SKYWIRE_NODE_PK")

	suite.T().Logf(`
	SKYWIRE_INTEGRATION_TESTS=%v 
	SKYWIRE_HOST=%v SKYWIRE_HOST_PK=%v 
	SKYWIRE_NODE=%v SKYWIRE_NODE_PK=%v`,
		envEnabled, suite.Host, suite.HostPK, suite.Node, suite.NodePK)
}

func TestTwoNodesSuite(t *testing.T) {
	suite.Run(t, new(TwoNodesSuite))
}

func (suite *TwoNodesSuite) Enabled() bool {
	if suite.Disabled {
		suite.T().Skip("Skywire tests are skipped")
		return false
	}
	return true
}

func sendmessage(nodeAddress, recipient, message string) (*http.Response, error) {
	data, _ := json.Marshal(map[string]string{"message": message, "recipient": recipient})
	return http.Post(nodeAddress, "application/json", bytes.NewReader(data))
}

func (suite *TwoNodesSuite) MessageToNode(message string) (*http.Response, error) {
	return sendmessage(suite.Host, suite.NodePK, message)
}

func (suite *TwoNodesSuite) MessageToHost(message string) (*http.Response, error) {
	return sendmessage(suite.Node, suite.HostPK, message)
}

// func (suite *TwoNodesSuite) TestMessageToHost(message string) {
// 	t := suite.T()
// 	if suite.Enabled() {
// 		resp, err := suite.MessageToHost("Disabled")
// 		require.Nil(t, err, "Got an error in MessageToHost")
// 		t.Logf("%v", resp)
// 	}
// }

func (suite *TwoNodesSuite) TstMessageToNode(message string) {
	t := suite.T()
	if suite.Enabled() {
		resp, err := suite.MessageToHost("B")
		// require.NoError(t, err, "Got an error in MessageToNode")
		t.Logf("%v %v", resp, err)
	}
}

func (suite *TwoNodesSuite) TestHelloMikeHelloJoe() {
	t := suite.T()
	if suite.Enabled() {
		resp, err := suite.MessageToNode("Hello Mike!")
		t.Logf("%v %v", resp, err)
		suite.MessageToHost("Hello Joe!")
		t.Logf("%v %v", resp, err)
		suite.MessageToNode("System is working!")
		t.Logf("%v %v", resp, err)
	}
}

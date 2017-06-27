package app

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/skycoin/skywire/src/apptracker"
	"github.com/skycoin/skywire/src/messages"
	"github.com/skycoin/skywire/src/node"
	network "github.com/skycoin/skywire/src/nodemanager"
)

var defaultConfig = &network.NodeManagerConfig{
	Domain:   "apps.network",
	CtrlAddr: "127.0.0.1:5999",
}

func TestCreateServer(t *testing.T) {
	messages.SetDebugLogLevel()

	appTrackerAddress := "127.0.0.1:14000"
	apptracker.NewAppTracker(appTrackerAddress)

	config := defaultConfig
	config.AppTrackerAddr = appTrackerAddress

	meshnet, _ := network.NewNetwork(defaultConfig)
	defer meshnet.Shutdown()

	serverNode, err := node.CreateNode(&node.NodeConfig{"127.0.0.1:5000", []string{"127.0.0.1:5999"}, 4000, ""})
	assert.Nil(t, err)
	defer serverNode.Shutdown()

	handle := func(in []byte) []byte {
		return in
	}

	server, err := NewServer(messages.MakeAppId("server1"), "127.0.0.1:4000", handle)
	assert.Nil(t, err)
	assert.Equal(t, messages.AppId([]byte("server1")), server.id)
	defer server.Shutdown()
}

func TestCreateClient(t *testing.T) {
	messages.SetDebugLogLevel()

	appTrackerAddress := "127.0.0.1:14001"
	apptracker.NewAppTracker(appTrackerAddress)

	config := defaultConfig
	config.AppTrackerAddr = appTrackerAddress

	meshnet, _ := network.NewNetwork(defaultConfig)
	defer meshnet.Shutdown()

	clientNode, err := node.CreateNode(&node.NodeConfig{"127.0.0.1:5000", []string{"127.0.0.1:5999"}, 4001, ""})
	assert.Nil(t, err)
	defer clientNode.Shutdown()

	client, err := NewClient(messages.MakeAppId("client1"), "127.0.0.1:4001")
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	assert.Equal(t, messages.AppId([]byte("client1")), client.id)
	defer client.Shutdown()
}

func TestSendWithFindRoute(t *testing.T) {
	messages.SetDebugLogLevel()

	appTrackerAddress := "127.0.0.1:14002"
	apptracker.NewAppTracker(appTrackerAddress)

	config := defaultConfig
	config.AppTrackerAddr = appTrackerAddress

	meshnet, _ := network.NewNetwork(defaultConfig)
	defer meshnet.Shutdown()

	clientNode, serverNode := meshnet.CreateThreeRoutes(14000)

	server, err := NewServer(messages.MakeAppId("server19"), serverNode.AppTalkAddr(), func(in []byte) []byte {
		return append(in, []byte("!!!")...)
	})
	assert.Nil(t, err)
	defer server.Shutdown()

	client, err := NewClient(messages.MakeAppId("client19"), clientNode.AppTalkAddr())
	assert.Nil(t, err)
	defer client.Shutdown()

	err = client.Connect(server.Id(), serverNode.Id().Hex())
	assert.Nil(t, err)

	response, err := client.Send([]byte("test"))

	assert.Nil(t, err)
	assert.Equal(t, "test!!!", string(response))
	time.Sleep(1 * time.Second)
}

func TestHandle(t *testing.T) {
	messages.SetInfoLogLevel()

	appTrackerAddress := "127.0.0.1:14003"
	apptracker.NewAppTracker(appTrackerAddress)

	config := defaultConfig
	config.AppTrackerAddr = appTrackerAddress

	meshnet, _ := network.NewNetwork(defaultConfig)
	defer meshnet.Shutdown()

	clientNode, serverNode := meshnet.CreateThreeRoutes(15000)

	server, err := NewServer(messages.MakeAppId("increasingServer"), serverNode.AppTalkAddr(), func(in []byte) []byte {
		size := len(in)
		result := make([]byte, size)
		for i := 0; i < size; i++ {
			result[i] = byte(i)
		}
		return result
	})
	assert.Nil(t, err)
	defer server.Shutdown()

	client, err := NewClient(messages.MakeAppId("Client of increasing server"), clientNode.AppTalkAddr())
	assert.Nil(t, err)
	defer client.Shutdown()

	err = client.Connect(server.Id(), serverNode.Id().Hex())
	assert.Nil(t, err)

	size := 100000

	request := make([]byte, size)

	response, err := client.Send(request)

	assert.Nil(t, err)
	assert.Len(t, response, size)

	correct := true
	for i := 0; i < size; i++ {
		if byte(i) != response[i] {
			correct = false
			break
		}
	}
	assert.True(t, correct)
	time.Sleep(1 * time.Second)
}

func TestSocks(t *testing.T) {
	messages.SetInfoLogLevel()

	appTrackerAddress := "127.0.0.1:14004"
	apptracker.NewAppTracker(appTrackerAddress)

	config := defaultConfig
	config.AppTrackerAddr = appTrackerAddress

	meshnet, _ := network.NewNetwork(defaultConfig)
	defer meshnet.Shutdown()

	clientNode, serverNode := meshnet.CreateSequenceOfNodes(20, 16000)

	client, err := NewSocksClient(messages.MakeAppId("socks client 0"), clientNode.AppTalkAddr(), "0.0.0.0:8000")
	assert.Nil(t, err)
	defer client.Shutdown()

	assert.Equal(t, client.ProxyAddress, "0.0.0.0:8000")

	server, err := NewSocksServer(messages.MakeAppId("socks server 0"), serverNode.AppTalkAddr(), "127.0.0.1:8001")
	assert.Nil(t, err)
	defer server.Shutdown()

	assert.Equal(t, server.ProxyAddress, "127.0.0.1:8001")

	err = client.Connect(server.Id(), "node20.apps.network")
	assert.Nil(t, err)
	time.Sleep(1 * time.Second)
}

func TestVPN(t *testing.T) {
	messages.SetInfoLogLevel()

	appTrackerAddress := "127.0.0.1:14005"
	apptracker.NewAppTracker(appTrackerAddress)

	config := defaultConfig
	config.AppTrackerAddr = appTrackerAddress

	meshnet, _ := network.NewNetwork(defaultConfig)
	defer meshnet.Shutdown()

	clientNode, serverNode := meshnet.CreateSequenceOfNodes(20, 17000)

	client, err := NewVPNClient(messages.MakeAppId("vpn_client"), clientNode.AppTalkAddr(), "0.0.0.0:4321")
	assert.Nil(t, err)
	defer client.Shutdown()
	assert.Equal(t, client.ProxyAddress, "0.0.0.0:4321")

	server, err := NewVPNServer(messages.MakeAppId("vpn_server"), serverNode.AppTalkAddr())
	assert.Nil(t, err)
	defer server.Shutdown()

	err = client.Connect(server.Id(), "node20.apps.network")
	assert.Nil(t, err)
}

func TestAppsRegistration(t *testing.T) {
	messages.SetInfoLogLevel()

	appTrackerAddress := "127.0.0.1:14006"
	apptracker.NewAppTracker(appTrackerAddress)

	config := defaultConfig
	config.AppTrackerAddr = appTrackerAddress

	meshnet, _ := network.NewNetwork(defaultConfig)
	defer meshnet.Shutdown()

	clientNode, serverNode := meshnet.CreateSequenceOfNodes(20, 18000)

	client, err := NewVPNClient(messages.MakeAppId("vpn_client_0"), clientNode.AppTalkAddr(), "0.0.0.0:4321")
	assert.Nil(t, err)
	defer client.Shutdown()
	assert.Equal(t, client.ProxyAddress, "0.0.0.0:4321")

	_, err = NewVPNServer(messages.MakeAppId("vpn_client_0"), serverNode.AppTalkAddr()) // the same name, should be an error
	assert.NotNil(t, err)
	log.Println(err)

	server, err := NewVPNServer(messages.MakeAppId("vpn_server_0"), serverNode.AppTalkAddr())
	assert.Nil(t, err)
	defer server.Shutdown()

	apps0, err := client.GetAppsList("all", "")
	assert.Nil(t, err)
	assert.Len(t, apps0, 2)

	apps1, err := client.GetAppsList("by_name", "vpn_client_0")
	assert.Nil(t, err)
	assert.Len(t, apps1, 1)

	apps2, err := client.GetAppsList("by_type", "vpn_server")
	assert.Nil(t, err)
	assert.Len(t, apps2, 1)

	apps3, err := client.GetAppsList("by_name", "non_existing_name")
	assert.Nil(t, err)
	assert.Len(t, apps3, 0)
}

package producer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"math/rand"
	"sync"
	"time"
)

var conf *Config
var sess *session.Session
var seq uint64 = 0
var discoveryName string
var discoveryKey string
var fieldMutex sync.RWMutex

func Init(path, dKey string) (err error) {
	conf = &Config{}
	err = LoadConfig(conf, path)
	if err != nil {
		return
	}
	if len(conf.AWSSecretKey) == 0 || len(conf.AWSAccessKeyId) == 0 || len(conf.QueueURL) == 0 || len(conf.Region) == 0 {
		err = fmt.Errorf("%s", "You need to fill in the correct information to ensure the discovery works.")
		panic(err)
	}
	sess, err = session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Credentials: credentials.NewStaticCredentials(conf.AWSAccessKeyId, conf.AWSSecretKey, ""),
			Region:      &conf.Region,
		},
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return
	}
	discoveryKey = dKey
	discoveryName = getRandomString()
	return
}

func Close() {
	conf = nil
	sess = nil
}

type MqBody struct {
	Key          string `json:"key"`
	Seq          uint64 `json:"seq"`
	FromApp      string `json:"from_app"`
	FromNode     string `json:"from_node"`
	ToNode       string `json:"to_node"`
	ToApp        string `json:"to_app"`
	Uid          uint64 `json:"uid"`
	FromHostPort string `json:"from_host_port"`
	ToHostPort   string `json:"to_host_port"`
	FromIp       string `json:"from_ip"`
	ToIp         string `json:"to_ip"`
	Count        uint64 `json:"count"`
	IsEnd        bool   `json:"is_end"`
}

type MqOnline struct {
	Key          string `json:"key"`
	Seq          uint64 `json:"seq"`
	Type         int    `json:"type"`
	NodeKey      string `json:"node_key"`
	DiscoveryKey string `json:"discovery_key"`
}


func Send(body *MqBody) (err error) {
	fieldMutex.Lock()
	seq++
	if seq == 0 {
		discoveryName = getRandomString()
		seq++
	}
	body.Key = discoveryName
	body.Seq = seq
	fieldMutex.Unlock()
	svc := sqs.New(sess)
	b, err := json.Marshal(&body)
	if err != nil {
		return
	}
	_, err = svc.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(b)),
		QueueUrl:    aws.String(conf.QueueURL),
	})
	return
}

func SendOnline(ol *MqOnline) (err error) {
	fieldMutex.Lock()
	seq++
	if seq == 0 {
		discoveryName = getRandomString()
		seq++
	}
	ol.DiscoveryKey = discoveryKey
	ol.Key = discoveryName
	ol.Seq = seq

	fieldMutex.Unlock()
	svc := sqs.New(sess)
	b, err := json.Marshal(&ol)
	if err != nil {
		return
	}
	_, err = svc.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(b)),
		QueueUrl:    aws.String(conf.OnlineQueueURL),
	})
	return
}


func getRandomString() string {
	bytes := make([]byte, 128)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Read(bytes)
	return hex.EncodeToString(bytes)
}

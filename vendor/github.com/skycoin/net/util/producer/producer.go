package producer

import (
	"encoding/hex"
	"encoding/json"
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
var fieldMutex sync.RWMutex

func Init(path string) (err error) {
	conf = &Config{}
	err = LoadConfig(conf, path)
	if err != nil {
		return
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
	if err != nil {
		return
	}
	return
}

func getRandomString() string {
	bytes := make([]byte, 128)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Read(bytes)
	return hex.EncodeToString(bytes)
}

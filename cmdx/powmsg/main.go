package main

import (
    "fmt"
    "log"
     "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/sqs"
    "time"
    "os"
    "flag"
)

const (
    Version = "0.1.0"
    Tag     = "dev"
)

var (
    qURL  string
    version bool
)

func parseFlags() {
    flag.StringVar(&qURL, "queue-url", "https://sqs.ap-northeast-1.amazonaws.com/589903953990/samosq", "sqs queue url")
    flag.BoolVar(&version, "v", false, "print current version")
    flag.Parse()
}

func main() {
    parseFlags()
    if version {
        fmt.Println(Version)
        return
    }

    file, err := os.OpenFile("./pow.log", os.O_CREATE|os.O_RDWR, 0666)
    defer file.Close();
    if err != nil {
        fmt.Println("Failed to log to file, using default stderr")
        return
    }
        logger := log.New(file,"",log.Ldate|log.Ltime);

    sess := session.Must(session.NewSessionWithOptions(session.Options{
        SharedConfigState: session.SharedConfigEnable,
    }))
    svc := sqs.New(sess)

    //qURL := "https://sqs.ap-northeast-1.amazonaws.com/589903953990/samosq"
    for true {
        result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
            AttributeNames: []*string{
                aws.String(sqs.MessageSystemAttributeNameApproximateFirstReceiveTimestamp),
                aws.String(sqs.MessageSystemAttributeNameSenderId),
                aws.String(sqs.MessageSystemAttributeNameApproximateReceiveCount),
                aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
            },
            MessageAttributeNames: []*string{
                aws.String(sqs.QueueAttributeNameAll),
            },
            QueueUrl:            &qURL,
            MaxNumberOfMessages: aws.Int64(10),
            VisibilityTimeout:   aws.Int64(36000), // 10 hours 36000
            WaitTimeSeconds:     aws.Int64(0),
        })

        if err != nil {
            fmt.Println("Error", err)
            return
        }
        if len(result.Messages) == 0 {
            fmt.Println("Received no messages")
            //return
        }else{
            logger.Println(result)
            //fmt.Println(result)
            fmt.Printf("Received %d messages\n",  len(result.Messages))
        }

        for i := 0; i < len(result.Messages); i++ {
            _, err := svc.DeleteMessage(&sqs.DeleteMessageInput{
                QueueUrl:      &qURL,
                ReceiptHandle: result.Messages[i].ReceiptHandle,
            })
            if err != nil {
                fmt.Println("Delete Error", err)
                 //return
            }
        }
        time.Sleep(5 * time.Second)

    }

}
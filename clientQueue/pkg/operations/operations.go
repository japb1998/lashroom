package operations

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type SQSClient struct {
	conn     *sqs.SQS
	QueueUrl *string
}

func NewSQSClient(queueName *string, queueUrl *string) (*SQSClient, error) {
	client := SQSClient{}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client.conn = sqs.New(sess)
	if queueUrl == nil {
		url, err := getQueueURL(client.conn, *queueName)
		if err != nil {
			return nil, fmt.Errorf("Error While Getting Queue Url: %v", err.Error())
		}
		*queueUrl = *url.QueueUrl
	}

	client.QueueUrl = queueUrl

	return &client, nil
}

func (c *SQSClient) SendMessage(messageBody []byte) error {
	_, err := c.conn.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    c.QueueUrl,
		MessageBody: aws.String(string(messageBody)),
	})

	return err
}

func (c *SQSClient) DeleteMessage(handle *string) error {
	_, err := c.conn.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      c.QueueUrl,
		ReceiptHandle: handle,
	})

	return err
}
func getQueueURL(sqsClient *sqs.SQS, queue string) (*sqs.GetQueueUrlOutput, error) {

	result, err := sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &queue,
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

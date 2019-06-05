package kettle

import (
	"fmt"
	"os"
	"strings"

	awszconf "github.com/NYTimes/gizmo/config/aws"
	zpubsub "github.com/NYTimes/gizmo/pubsub"
	awszpubsub "github.com/NYTimes/gizmo/pubsub/aws"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/dchest/uniuri"
)

type Creds struct {
	Region string
	Key    string
	Secret string
}

func getAcctId(c ...Creds) (*string, error) {
	var sess *session.Session
	var err error
	region := os.Getenv("AWS_REGION")
	if len(c) > 0 {
		if c[0].Region != "" {
			region = c[0].Region
		}

		if c[0].Key != "" && c[0].Secret != "" {
			sess, err = session.NewSession(&aws.Config{
				Region:      aws.String(region),
				Credentials: credentials.NewStaticCredentials(c[0].Key, c[0].Secret, ""),
			})
		}
	}

	if sess == nil {
		sess = session.Must(session.NewSession())
	}

	svc := sts.New(sess)
	res, err := svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	return res.Account, nil
}

// getSqsAllowAllPolicy returns a policy that can be used when creating an SQS queue that allow
// all SQS actions for everybody.
func getSqsAllowAllPolicy(region, acct, queueName string) string {
	return `{
  "Version":"2008-10-17",
  "Id":"id` + strings.ToLower(uniuri.NewLen(10)) + `",
  "Statement":[
    {
	  "Sid":"sid` + strings.ToLower(uniuri.NewLen(10)) + `",
	  "Effect":"Allow",
	  "Principal":"*",
	  "Action":"SQS:*",
	  "Resource":"` + fmt.Sprintf("arn:aws:sqs:%s:%s:%s", region, acct, queueName) + `"
    }
  ]
}`
}

// GetTopic returns the ARN of a newly created topic or an existing one. CreateTopic API
// returns the ARN of an existing topic.
func GetTopic(name string, c ...Creds) (*string, error) {
	var sess *session.Session
	var err error
	region := os.Getenv("AWS_REGION")
	if len(c) > 0 {
		if c[0].Region != "" {
			region = c[0].Region
		}

		if c[0].Key != "" && c[0].Secret != "" {
			sess, err = session.NewSession(&aws.Config{
				Region:      aws.String(region),
				Credentials: credentials.NewStaticCredentials(c[0].Key, c[0].Secret, ""),
			})
		}
	}

	if sess == nil {
		sess = session.Must(session.NewSession())
	}

	svc := sns.New(sess)
	res, err := svc.CreateTopic(&sns.CreateTopicInput{
		Name: aws.String(name),
	})

	if err != nil {
		return nil, err
	}

	return res.TopicArn, nil
}

func NewPublisher(name string, c ...Creds) (zpubsub.Publisher, error) {
	topicArn, err := GetTopic(name, c...)
	if err != nil {
		return nil, err
	}

	region := os.Getenv("AWS_REGION")
	key := os.Getenv("AWS_ACCESS_KEY_ID")
	secret := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if len(c) > 0 {
		if c[0].Region != "" {
			region = c[0].Region
		}

		if c[0].Key != "" {
			key = c[0].Key
		}

		if c[0].Secret != "" {
			secret = c[0].Secret
		}
	}

	cnf := awszpubsub.SNSConfig{
		Config: awszconf.Config{
			Region:    region,
			AccessKey: key,
			SecretKey: secret,
		},
		Topic: *topicArn,
	}

	return awszpubsub.NewPublisher(cnf)
}

// GetSqs creates an SQS queue and returning the queue url and attributes.
func GetSqs(name string, c ...Creds) (*string, map[string]*string, error) {
	var sess *session.Session
	var err error
	region := os.Getenv("AWS_REGION")
	if len(c) > 0 {
		if c[0].Region != "" {
			region = c[0].Region
		}

		if c[0].Key != "" && c[0].Secret != "" {
			sess, err = session.NewSession(&aws.Config{
				Region:      aws.String(region),
				Credentials: credentials.NewStaticCredentials(c[0].Key, c[0].Secret, ""),
			})
		}
	}

	if sess == nil {
		sess = session.Must(session.NewSession())
	}

	svc := sqs.New(sess)

	var acct *string
	acct, err = getAcctId(c...)
	if err != nil {
		return nil, nil, err
	}

	policy := getSqsAllowAllPolicy(region, *acct, name)
	create, err := svc.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(name),
		Attributes: map[string]*string{
			"Policy": aws.String(policy),
		},
	})

	if err != nil {
		return nil, nil, err
	}

	qAttr, err := svc.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl: create.QueueUrl,
		AttributeNames: []*string{
			aws.String("All"),
		},
	})

	if err != nil {
		return nil, nil, err
	}

	return create.QueueUrl, qAttr.Attributes, nil
}

// SubscribeToTopic creates the queue, or use an existing queue, and subscribe to the
// provided SNS topic.
func SubscribeToTopic(name, topicArn string, c ...Creds) error {
	var sess *session.Session
	var err error
	region := os.Getenv("AWS_REGION")
	if len(c) > 0 {
		if c[0].Region != "" {
			region = c[0].Region
		}

		if c[0].Key != "" && c[0].Secret != "" {
			sess, err = session.NewSession(&aws.Config{
				Region:      aws.String(region),
				Credentials: credentials.NewStaticCredentials(c[0].Key, c[0].Secret, ""),
			})
		}
	}

	if sess == nil {
		sess = session.Must(session.NewSession())
	}

	_, qattr, err := GetSqs(name, c...)
	if err != nil {
		return err
	}

	svc := sns.New(sess)
	_, err = svc.Subscribe(&sns.SubscribeInput{
		TopicArn: aws.String(topicArn),
		Protocol: aws.String("sqs"),
		Endpoint: qattr["QueueArn"],
		Attributes: map[string]*string{
			"RawMessageDelivery": aws.String("true"),
		},
	})

	return err
}

func NewSubscriber(name string, c ...Creds) (zpubsub.Subscriber, error) {
	acct, err := getAcctId(c...)
	if err != nil {
		return nil, err
	}

	region := os.Getenv("AWS_REGION")
	if len(c) > 0 {
		if c[0].Region != "" {
			region = c[0].Region
		}
	}

	return awszpubsub.NewSubscriber(awszpubsub.SQSConfig{
		Config: awszconf.Config{
			Region: region,
		},
		QueueName:           name,
		QueueOwnerAccountID: *acct,
		ConsumeBase64:       aws.Bool(false),
	})
}

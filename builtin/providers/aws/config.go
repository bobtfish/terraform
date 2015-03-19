package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/autoscaling"
	"github.com/hashicorp/aws-sdk-go/gen/ec2"
	"github.com/hashicorp/aws-sdk-go/gen/elb"
	"github.com/hashicorp/aws-sdk-go/gen/rds"
	"github.com/hashicorp/aws-sdk-go/gen/route53"
	"github.com/hashicorp/aws-sdk-go/gen/s3"
	"github.com/hashicorp/terraform/helper/multierror"
)

type Config struct {
	AccessKey              string
	SecretKey              string
	Token                  string
	CredentialsFilePath    string
	CredentialsFileProfile string
	Region                 string
	Provider               aws.CredentialsProvider
}

type AWSClient struct {
	ec2conn         *ec2.EC2
	elbconn         *elb.ELB
	autoscalingconn *autoscaling.AutoScaling
	s3conn          *s3.S3
	r53conn         *route53.Route53
	region          string
	rdsconn         *rds.RDS
}

func (c *Config) loadAndValidate(providerCode string) (interface{}, error) {
	credsProvider, err := c.getCredsProvider(providerCode)
	if err != nil {
		return nil, err
	}

	if _, err := credsProvider.Credentials(); err != nil {
		return nil, err
	}

	c.Provider = credsProvider

	return c.Client()
}

func (c *Config) getCredsProvider(providerCode string) (aws.CredentialsProvider, error) {
	if providerCode == "static" {
		return aws.Creds(c.AccessKey, c.SecretKey, c.Token), nil
	} else if providerCode == "iam" {
		return aws.IAMCreds(), nil
	} else if providerCode == "env" {
		return aws.EnvCreds()
	} else if providerCode == "file" {
		// TODO: Could be a variable but there's no standardized name for it
		// More importantly, what is really the point of this variable??
		expiry := 10 * time.Minute

		return aws.ProfileCreds(
			c.CredentialsFilePath, c.CredentialsFileProfile, expiry)
	}
	return aws.DetectCreds(c.AccessKey, c.SecretKey, c.Token), nil
}

// Client configures and returns a fully initailized AWSClient
func (c *Config) Client() (interface{}, error) {
	var client AWSClient

	// Get the auth and region. This can fail if keys/regions were not
	// specified and we're attempting to use the environment.
	var errs []error

	log.Println("[INFO] Building AWS region structure")
	err := c.ValidateRegion()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		// store AWS region in client struct, for region specific operations such as
		// bucket storage in S3
		client.region = c.Region
		credsProvider := c.Provider

		log.Println("[INFO] Initializing ELB connection")
		client.elbconn = elb.New(credsProvider, c.Region, nil)
		log.Println("[INFO] Initializing AutoScaling connection")
		client.autoscalingconn = autoscaling.New(credsProvider, c.Region, nil)
		log.Println("[INFO] Initializing S3 connection")
		client.s3conn = s3.New(credsProvider, c.Region, nil)
		log.Println("[INFO] Initializing RDS connection")
		client.rdsconn = rds.New(credsProvider, c.Region, nil)

		// aws-sdk-go uses v4 for signing requests, which requires all global
		// endpoints to use 'us-east-1'.
		// See http://docs.aws.amazon.com/general/latest/gr/sigv4_changes.html
		log.Println("[INFO] Initializing Route53 connection")
		client.r53conn = route53.New(credsProvider, "us-east-1", nil)
		log.Println("[INFO] Initializing EC2 Connection")
		client.ec2conn = ec2.New(credsProvider, c.Region, nil)
	}

	if len(errs) > 0 {
		return nil, &multierror.Error{Errors: errs}
	}

	return &client, nil
}

// IsValidRegion returns true if the configured region is a valid AWS
// region and false if it's not
func (c *Config) ValidateRegion() error {
	var regions = [11]string{"us-east-1", "us-west-2", "us-west-1", "eu-west-1",
		"eu-central-1", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
		"sa-east-1", "cn-north-1", "us-gov-west-1"}

	for _, valid := range regions {
		if c.Region == valid {
			return nil
		}
	}
	return fmt.Errorf("Not a valid region: %s", c.Region)
}

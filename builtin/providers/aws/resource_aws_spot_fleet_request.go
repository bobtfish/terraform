package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSpotFleetRequest() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSpotFleetRequestCreate,
		Read:   resourceAwsSpotFleetRequestRead,
		Delete: resourceAwsSpotFleetRequestDelete,
		Update: resourceAwsSpotFleetRequestUpdate,

		Schema: map[string]*schema.Schema{
			"iam_fleet_role": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"launch_specifications": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				// TODO: Figure out what the right schema is for this list
				Elem: &schema.Schema{Type: schema.TypeString},
			},
			"spot_price": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			// Everything on a spot fleet is ForceNew except target_capacity
			"target_capacity": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"allocation_strategy": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
			},
			"excess_capacity_termination_policy": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
			},
			"terminate_instances_with_expiration": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"valid_from": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
			},
			"valid_until": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsSpotFleetRequestCreate(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_RequestSpotFleet.html
	conn := meta.(*AWSClient).ec2conn

	// http://docs.aws.amazon.com/sdk-for-go/api/service/ec2.html#type-SpotFleetRequestConfigData
	spotFleetConfig := &ec2.SpotFleetRequestConfigData{
		IamFleetRole:                     aws.String(d.Get("iam_fleet_role").(string)),
		LaunchSpecifications:             []*ec2.SpotFleetLaunchSpecification{}, // TODO: Read this
		SpotPrice:                        aws.String(d.Get("spot_price").(string)),
		TargetCapacity:                   aws.Int64(int64(d.Get("target_capacity").(int))),
		AllocationStrategy:               aws.String(d.Get("allocation_strategy").(string)),
		ClientToken:                      aws.String(resource.UniqueId()),
		ExcessCapacityTerminationPolicy:  aws.String(d.Get("excess_capacity_termination_policy").(string)),
		TerminateInstancesWithExpiration: aws.Bool(d.Get("terminate_instances_with_expiration").(bool)),
		ValidFrom:                        aws.Time(time.Now()), // TODO: Read time?
		ValidUntil:                       aws.Time(time.Now()), // TODO: Read time?
	}

	// http://docs.aws.amazon.com/sdk-for-go/api/service/ec2.html#type-RequestSpotFleetInput
	spotFleetOpts := &ec2.RequestSpotFleetInput{
		SpotFleetRequestConfig: spotFleetConfig,
		DryRun:                 aws.Bool(false),
	}

	log.Printf("[DEBUG] Requesting spot fleet with these opts: %s", spotFleetOpts)
	resp, err := conn.RequestSpotFleet(spotFleetOpts)
	if err != nil {
		return fmt.Errorf("Error requesting spot fleet: %s", err)
	}

	d.SetId(*resp.SpotFleetRequestId)

	return resourceAwsSpotFleetRequestUpdate(d, meta)
}

func resourceAwsSpotFleetRequestRead(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeSpotFleetRequests.html
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeSpotFleetRequestsInput{
		SpotFleetRequestIds: []*string{aws.String(d.Id())},
	}
	resp, err := conn.DescribeSpotFleetRequests(req)

	if err != nil {
		// If the spot request was not found, return nil so that we can show
		// that it is gone.
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidSpotFleetRequestID.NotFound" {
			d.SetId("")
			return nil
		}

		// Some other error, report it
		return err
	}

	request := resp.SpotFleetRequestConfigs[0]

	// if the request is cancelled, then it is gone
	if *request.SpotFleetRequestState == "cancelled" {
		d.SetId("")
		return nil
	}

	d.Set("spot_request_state", request.SpotFleetRequestState)
	return nil
}

func resourceAwsSpotFleetRequestUpdate(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_ModifySpotFleetRequest.html
	conn := meta.(*AWSClient).ec2conn

	d.Partial(true)
	// TODO: Adjust target capacity
	if err := setTags(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsSpotFleetRequestRead(d, meta)
}

func resourceAwsSpotFleetRequestDelete(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CancelSpotFleetRequests.html
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[INFO] Cancelling spot fleet request: %s", d.Id())
	_, err := conn.CancelSpotFleetRequests(&ec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: []*string{aws.String(d.Id())},
	})

	if err != nil {
		return fmt.Errorf("Error cancelling spot request (%s): %s", d.Id(), err)
	}

	// TODO: Should we terminate or let the spot fleet policy do that?
	return nil
}

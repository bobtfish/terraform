package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
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
			// http://docs.aws.amazon.com/sdk-for-go/api/service/ec2.html#type-SpotFleetLaunchSpecification
			"launch_specifications": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				// TODO: Figure out what the right schema is for this list
				Elem: &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
					return hashcode.String(buf.String())
				},
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
				Optional: true,
				ForceNew: true,
			},
			"excess_capacity_termination_policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"terminate_instances_with_expiration": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"valid_from": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"valid_until": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func buildAwsSpotFleetLaunchSpecification(
	d *schema.ResourceData, meta interface{}) ([]*ec2.SpotFleetLaunchSpecification, error) {

	specs := []*ec2.SpotFleetLaunchSpecification{}
	user_specs := d.Get("launch_specifications").(*schema.Set).List()
	for _, user_spec := range user_specs {
		user_spec_map := user_spec.(map[string]interface{})
		specs = append(specs, &ec2.SpotFleetLaunchSpecification{
			// TODO: Yuck
			BlockDeviceMappings: user_spec_map["block_device_mappings"],
			EbsOptimized:        aws.Bool(user_spec_map["ebs_optimized"].(bool)),
			IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
				Name: aws.String(user_spec_map["iam_instance_profile"].(string)),
			},
			ImageId:      aws.String(user_spec_map["ami"].(string)),
			InstanceType: aws.String(user_spec_map["instance_type"].(string)),
			KernelId:     aws.String(user_spec_map["kernel_id"].(string)),
			KeyName:      aws.String(user_spec_map["key_name"].(string)),
			Monitoring: &ec2.SpotFleetMonitoring{
				Enabled: aws.Bool(user_spec_map["monitoring"].(bool)),
			},
			// TODO: Yuck
			NetworkInterfaces: aws.String(user_spec_map["network_interfaces"].(string)),
			Placement: &ec2.SpotPlacement{
				AvailabilityZone: aws.String(user_spec_map["availability_zone"].(string)),
			},
			RamdiskId: aws.String(user_spec_map["ram_disk_id"].(string)),
			// TODO: Yuck
			SecurityGroups: aws.String(user_spec_map["security_groups"].(string)),
			SpotPrice:      aws.String(user_spec_map["spot_price"].(string)),
			SubnetId:       aws.String(user_spec_map["subnet_id"].(string)),
			UserData:       aws.String(user_spec_map["user_data"].(string)),
		})
	}
	return specs, nil
}

func resourceAwsSpotFleetRequestCreate(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_RequestSpotFleet.html
	conn := meta.(*AWSClient).ec2conn

	launch_specs, err := buildAwsSpotFleetLaunchSpecification(d, meta)
	if err != nil {
		return err
	}

	// http://docs.aws.amazon.com/sdk-for-go/api/service/ec2.html#type-SpotFleetRequestConfigData
	spotFleetConfig := &ec2.SpotFleetRequestConfigData{
		IamFleetRole:                     aws.String(d.Get("iam_fleet_role").(string)),
		LaunchSpecifications:             launch_specs,
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
	//	conn := meta.(*AWSClient).ec2conn

	d.Partial(true)
	// TODO: Adjust target capacity

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

	return nil
}

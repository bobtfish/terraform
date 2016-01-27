package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
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
			// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_SpotFleetLaunchSpecification.html
			"launch_specification": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ebs_block_device": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"delete_on_termination": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
										Default:  true,
										ForceNew: true,
									},
									"device_name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
									},
									"encrypted": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"iops": &schema.Schema{
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"snapshot_id": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"volume_size": &schema.Schema{
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"volume_type": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
								},
							},
							Set: func(v interface{}) int {
								var buf bytes.Buffer
								m := v.(map[string]interface{})
								buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
								buf.WriteString(fmt.Sprintf("%s-", m["snapshot_id"].(string)))
								return hashcode.String(buf.String())
							},
						},
						"ephemeral_block_device": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							ForceNew: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"device_name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"virtual_name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							Set: func(v interface{}) int {
								var buf bytes.Buffer
								m := v.(map[string]interface{})
								buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
								buf.WriteString(fmt.Sprintf("%s-", m["virtual_name"].(string)))
								return hashcode.String(buf.String())
							},
						},
						"root_block_device": &schema.Schema{
							// TODO: This is a set because we don't support singleton
							//       sub-resources today. We'll enforce that the set only ever has
							//       length zero or one below. When TF gains support for
							//       sub-resources this can be converted.
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								// "You can only modify the volume size, volume type, and Delete on
								// Termination flag on the block device mapping entry for the root
								// device volume." - bit.ly/ec2bdmap
								Schema: map[string]*schema.Schema{
									"delete_on_termination": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
										Default:  true,
										ForceNew: true,
									},
									"iops": &schema.Schema{
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"volume_size": &schema.Schema{
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"volume_type": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
								},
							},
							Set: func(v interface{}) int {
								// there can be only one root device; no need to hash anything
								return 0
							},
						},
						"ebs_optimized": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"iam_instance_profile": &schema.Schema{
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
						},
						"ami": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"instance_type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"key_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
						"monitoring": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						//									"network_interface_set"
						"placement_group": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
						"spot_price": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"subnet_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
						"user_data": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							StateFunc: func(v interface{}) string {
								switch v.(type) {
								case string:
									hash := sha1.Sum([]byte(v.(string)))
									return hex.EncodeToString(hash[:])
								default:
									return ""
								}
							},
						},
						"weighted_capacity": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						// TODO double check this
						"availability_zone": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["availability_zone"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["instance_type"].(string)))
					return hashcode.String(buf.String())
				},
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
			"spot_price": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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
	user_specs := d.Get("launch_specification").(*schema.Set).List()
	for _, user_spec := range user_specs {
		user_spec_map := user_spec.(map[string]interface{})
		// panic: interface conversion: interface {} is map[string]interface {}, not *schema.ResourceData
		instanceOpts, err := buildAwsInstanceOpts(user_spec.(*schema.ResourceData), meta)
		if err != nil {
			return nil, err
		}
		specs = append(specs, &ec2.SpotFleetLaunchSpecification{
			BlockDeviceMappings: instanceOpts.BlockDeviceMappings,
			EbsOptimized:        instanceOpts.EBSOptimized,
			IamInstanceProfile:  instanceOpts.IAMInstanceProfile,
			ImageId:             instanceOpts.ImageID,
			InstanceType:        instanceOpts.InstanceType,
			KeyName:             instanceOpts.KeyName,
			Monitoring: &ec2.SpotFleetMonitoring{
				Enabled: aws.Bool(user_spec_map["monitoring"].(bool)),
			},
			NetworkInterfaces: instanceOpts.NetworkInterfaces,
			Placement:         instanceOpts.SpotPlacement,
			SubnetId:          instanceOpts.SubnetID,
			UserData:          instanceOpts.UserData64,
			SpotPrice:         aws.String(user_spec_map["spot_price"].(string)),
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
		TerminateInstancesWithExpiration: aws.Bool(d.Get("terminate_instances_with_expiration").(bool)),
		ValidFrom:                        aws.Time(time.Now()), // TODO: Read time?
		ValidUntil:                       aws.Time(time.Now()), // TODO: Read time?
	}

	if v, ok := d.GetOk("excess_capacity_termination_policy"); ok {
		spotFleetConfig.ExcessCapacityTerminationPolicy = aws.String(v.(string))
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

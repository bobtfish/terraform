package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/autoscaling"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLaunchConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLaunchConfigurationCreate,
		Read:   resourceAwsLaunchConfigurationRead,
		Delete: resourceAwsLaunchConfigurationDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"iam_instance_profile": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"key_name": &schema.Schema{
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

			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"associate_public_ip_address": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"spot_price": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"block_device": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"virtual_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"snapshot_id": &schema.Schema{
							Type:     schema.TypeString,
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

						"volume_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"delete_on_termination": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
							ForceNew: true,
						},

						"encrypted": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%t-", m["delete_on_termination"].(bool)))
					buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
					// See the NOTE in "ebs_block_device" for why we skip iops here.
					// buf.WriteString(fmt.Sprintf("%d-", m["iops"].(int)))
					buf.WriteString(fmt.Sprintf("%d-", m["volume_size"].(int)))
					buf.WriteString(fmt.Sprintf("%s-", m["volume_type"].(string)))
					return hashcode.String(buf.String())
				},
			},
		},
	}
}

func resourceAwsLaunchConfigurationCreate(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	var createLaunchConfigurationOpts autoscaling.CreateLaunchConfigurationType
	createLaunchConfigurationOpts.LaunchConfigurationName = aws.String(d.Get("name").(string))
	createLaunchConfigurationOpts.ImageID = aws.String(d.Get("image_id").(string))
	createLaunchConfigurationOpts.InstanceType = aws.String(d.Get("instance_type").(string))

	if v, ok := d.GetOk("user_data"); ok {
		createLaunchConfigurationOpts.UserData = aws.String(base64.StdEncoding.EncodeToString([]byte(v.(string))))
	}
	if v, ok := d.GetOk("associate_public_ip_address"); ok {
		createLaunchConfigurationOpts.AssociatePublicIPAddress = aws.Boolean(v.(bool))
	}
	if v, ok := d.GetOk("iam_instance_profile"); ok {
		createLaunchConfigurationOpts.IAMInstanceProfile = aws.String(v.(string))
	}
	if v, ok := d.GetOk("key_name"); ok {
		createLaunchConfigurationOpts.KeyName = aws.String(v.(string))
	}
	if v, ok := d.GetOk("spot_price"); ok {
		createLaunchConfigurationOpts.SpotPrice = aws.String(v.(string))
	}

	if v, ok := d.GetOk("security_groups"); ok {
		createLaunchConfigurationOpts.SecurityGroups = expandStringList(
			v.(*schema.Set).List())
	}

	if v := d.Get("block_device"); v != nil {
		vs := v.(*schema.Set).List()
		if len(vs) > 0 {
			createLaunchConfigurationOpts.BlockDeviceMappings = make([]autoscaling.BlockDeviceMapping, len(vs))
			for i, v := range vs {
				bd := v.(map[string]interface{})
				createLaunchConfigurationOpts.BlockDeviceMappings[i].DeviceName = bd["device_name"].(aws.StringValue)
				createLaunchConfigurationOpts.BlockDeviceMappings[i].VirtualName = bd["virtual_name"].(aws.StringValue)
				createLaunchConfigurationOpts.BlockDeviceMappings[i].EBS.SnapshotID = bd["snapshot_id"].(aws.StringValue)
				createLaunchConfigurationOpts.BlockDeviceMappings[i].EBS.VolumeType = bd["volume_type"].(aws.StringValue)
				createLaunchConfigurationOpts.BlockDeviceMappings[i].EBS.VolumeSize = bd["volume_size"].(aws.IntegerValue)
				createLaunchConfigurationOpts.BlockDeviceMappings[i].EBS.DeleteOnTermination = bd["delete_on_termination"].(aws.BooleanValue)
			}
		}
	}
	if v, ok := d.GetOk("name"); ok {
		createLaunchConfigurationOpts.LaunchConfigurationName = aws.String(v.(string))
		d.SetId(d.Get("name").(string))
	} else {
		hash := sha1.Sum([]byte(fmt.Sprintf("%#v", createLaunchConfigurationOpts)))
		config_name := fmt.Sprintf("terraform-%s", base64.URLEncoding.EncodeToString(hash[:]))
		log.Printf("[DEBUG] Computed Launch config name: %s", config_name)
		createLaunchConfigurationOpts.LaunchConfigurationName = aws.String(config_name)
		d.SetId(config_name)
	}

	log.Printf("[DEBUG] autoscaling create launch configuration: %#v", createLaunchConfigurationOpts)
	err := autoscalingconn.CreateLaunchConfiguration(&createLaunchConfigurationOpts)
	if err != nil {
		return fmt.Errorf("Error creating launch configuration: %s", err)
	}

	log.Printf("[INFO] launch configuration ID: %s", d.Id())

	// We put a Retry here since sometimes eventual consistency bites
	// us and we need to retry a few times to get the LC to load properly
	return resource.Retry(30*time.Second, func() error {
		return resourceAwsLaunchConfigurationRead(d, meta)
	})
}

func resourceAwsLaunchConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	describeOpts := autoscaling.LaunchConfigurationNamesType{
		LaunchConfigurationNames: []string{d.Id()},
	}

	log.Printf("[DEBUG] launch configuration describe configuration: %#v", describeOpts)
	describConfs, err := autoscalingconn.DescribeLaunchConfigurations(&describeOpts)
	if err != nil {
		return fmt.Errorf("Error retrieving launch configuration: %s", err)
	}
	if len(describConfs.LaunchConfigurations) == 0 {
		d.SetId("")
		return nil
	}

	// Verify AWS returned our launch configuration
	if *describConfs.LaunchConfigurations[0].LaunchConfigurationName != d.Id() {
		return fmt.Errorf(
			"Unable to find launch configuration: %#v",
			describConfs.LaunchConfigurations)
	}

	lc := describConfs.LaunchConfigurations[0]

	d.Set("key_name", *lc.KeyName)
	d.Set("image_id", *lc.ImageID)
	d.Set("instance_type", *lc.InstanceType)
	d.Set("name", *lc.LaunchConfigurationName)

	bds := make([]map[string]interface{}, len(lc.BlockDeviceMappings))
	for i, m := range lc.BlockDeviceMappings {
		bds[i] = make(map[string]interface{})
		bds[i]["device_name"] = m.DeviceName
		bds[i]["snapshot_id"] = m.EBS.SnapshotID
		bds[i]["volume_type"] = m.EBS.VolumeType
		bds[i]["volume_size"] = m.EBS.VolumeSize
		bds[i]["delete_on_termination"] = m.EBS.DeleteOnTermination
	}
	d.Set("block_device", bds)

	if lc.IAMInstanceProfile != nil {
		d.Set("iam_instance_profile", *lc.IAMInstanceProfile)
	} else {
		d.Set("iam_instance_profile", nil)
	}

	if lc.SpotPrice != nil {
		d.Set("spot_price", *lc.SpotPrice)
	} else {
		d.Set("spot_price", nil)
	}

	if lc.SecurityGroups != nil {
		d.Set("security_groups", lc.SecurityGroups)
	} else {
		d.Set("security_groups", nil)
	}

	return nil
}

func resourceAwsLaunchConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	log.Printf("[DEBUG] Launch Configuration destroy: %v", d.Id())
	err := autoscalingconn.DeleteLaunchConfiguration(
		&autoscaling.LaunchConfigurationNameType{LaunchConfigurationName: aws.String(d.Id())})
	if err != nil {
		autoscalingerr, ok := err.(aws.APIError)
		if ok && autoscalingerr.Code == "InvalidConfiguration.NotFound" {
			return nil
		}

		return err
	}

	return nil
}

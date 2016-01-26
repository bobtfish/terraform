---
layout: "aws"
page_title: "AWS: aws_spot_fleet_request"
sidebar_current: "docs-aws-resource-spot-fleet-request"
description: |-
  Provides a Spot Fleet Request resource.
---

# aws\_spot\_fleet\_request

// TODO
// https://www.terraform.io/docs/providers/aws/r/spot_instance_request.html

Provides an EC2 Spot Fleet Request resource. This allows a fleetinstances to be
requested on the spot market.

## Example Usage

// TODO
```
# Request a spot fleet instance
resource "aws_spot_fleet_request" "myfleet" {
    ami = "ami-1234"
    spot_price = "0.03"
    instance_type = "c4.xlarge"
}
```

## Argument Reference

// TODO
Spot Instance Requests support all the same arguments as
[`aws_instance`](instance.html), with the addition of:

* `spot_price` - (Required) The price to request on the spot market.
* `wait_for_fulfillment` - (Optional; Default: false) If set, Terraform will
  wait for the Spot Request to be fulfilled, and will throw an error if the
  timeout of 10m is reached.
* `spot_type` - (Optional; Default: "persistent") If set to "one-time", after
  the instance is terminated, the spot request will be closed. Also, Terraform
  can't manage one-time spot requests, just launch them.
* `block_duration_minutes` - (Optional) The required duration for the Spot instances, in minutes. This value must be a multiple of 60 (60, 120, 180, 240, 300, or 360).
  The duration period starts as soon as your Spot instance receives its instance ID. At the end of the duration period, Amazon EC2 marks the Spot instance for termination and provides a Spot instance termination notice, which gives the instance a two-minute warning before it terminates.
  Note that you can't specify an Availability Zone group or a launch group if you specify a duration.

## Attributes Reference

// TODO
The following attributes are exported:

* `id` - The Spot Instance Request ID.

These attributes are exported, but they are expected to change over time and so
should only be used for informational purposes, not for resource dependencies:

* `spot_bid_status` - The current [bid
  status](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-bid-status.html)
  of the Spot Instance Request.
* `spot_request_state` The current [request
  state](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-requests.html#creating-spot-request-status)
  of the Spot Instance Request.
* `spot_instance_id` - The Instance ID (if any) that is currently fulfilling
  the Spot Instance request.
* `public_dns` - The public DNS name assigned to the instance. For EC2-VPC, this 
  is only available if you've enabled DNS hostnames for your VPC
* `public_ip` - The public IP address assigned to the instance, if applicable.
* `private_dns` - The private DNS name assigned to the instance. Can only be 
  used inside the Amazon EC2, and only available if you've enabled DNS hostnames 
  for your VPC
* `private_ip` - The private IP address assigned to the instance

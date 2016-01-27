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

# Request a simple spot fleet
```
resource "aws_spot_fleet_request" "myfleet" {
    iam_fleet_role = 'foo'
    launch_specification {
        instance_type = "m3.xlarge"
        bid_price = "$1"
        weight = 2
        ...block_device {
        ...}

        network_interface {
            associate_public_ipaddress = false
            deleter_on_tgermination = true
        }

        securitygroups = [ "default" ]
    }
    launch_specification {
        instance_type = "m3.large"
        bid_price = "$1"
        weight = 1
    }
    spot_price = '0.01'
}
```

## Argument Reference

Most of these arguments directly correspond to the
[offical API](http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_SpotFleetRequestConfigData.html).

* `iam_fleet_role` - (required) Grants the Spot fleet permission to terminate
Spot instances on your behalf when you cancel its Spot fleet request using
CancelSpotFleetRequests or when the Spot fleet request expires, if you set
terminateInstancesWithExpiration.
* `launch_specifications` - TODO
* `spot_price` - (required) The bid price per unit hour.
* `target_capacity` - The number of units to request. You can choose to set the
  target capacity in terms of instances or a performance characteristic that is
important to your application workload, such as vCPUs, memory, or I/O.
* `allocation_strategy` - Indicates how to allocate the target capacity across
  the Spot pools specified by the Spot fleet request. The default is
lowestPrice.
* `excess_capacity_termination_policy` - Indicates whether running Spot
  instances should be terminated if the target capacity of the Spot fleet
request is decreased below the current size of the Spot fleet. 
* `terminate_instances_with_expiration` -Indicates whether running Spot
  instances should be terminated when the Spot fleet request expires.
* `valid_from` - The start date and time of the request, in UTC format (for
  example, YYYY-MM-DDTHH:MM:SSZ). The default is to start fulfilling the
request immediately. 
* `valid_until` - The end date and time of the request, in UTC format (for
  example, YYYY-MM-DDTHH:MM:SSZ). At this point, no new Spot instance requests
are placed or enabled to fulfill the request.


## Attributes Reference

The following attributes are exported:

* `id` - The Spot Instance Request ID.

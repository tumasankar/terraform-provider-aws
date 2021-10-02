package ec2

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func DataSourceRouteTable() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceRouteTableRead,

		Schema: map[string]*schema.Schema{
			"subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"gateway_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"route_table_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"filter": ec2CustomFiltersSchema(),
			"tags":   tftags.TagsSchemaComputed(),
			"routes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						///
						// Destinations.
						///
						"cidr_block": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"ipv6_cidr_block": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"destination_prefix_list_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						///
						// Targets.
						///
						"carrier_gateway_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"egress_only_gateway_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"gateway_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"instance_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"local_gateway_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"nat_gateway_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"network_interface_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"transit_gateway_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"vpc_endpoint_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"vpc_peering_connection_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"associations": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"route_table_association_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"route_table_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"subnet_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"gateway_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"main": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceRouteTableRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	req := &ec2.DescribeRouteTablesInput{}
	vpcId, vpcIdOk := d.GetOk("vpc_id")
	subnetId, subnetIdOk := d.GetOk("subnet_id")
	gatewayId, gatewayIdOk := d.GetOk("gateway_id")
	rtbId, rtbOk := d.GetOk("route_table_id")
	tags, tagsOk := d.GetOk("tags")
	filter, filterOk := d.GetOk("filter")

	if !rtbOk && !vpcIdOk && !subnetIdOk && !gatewayIdOk && !filterOk && !tagsOk {
		return fmt.Errorf("one of route_table_id, vpc_id, subnet_id, gateway_id, filters, or tags must be assigned")
	}
	req.Filters = BuildAttributeFilterList(
		map[string]string{
			"route-table-id":         rtbId.(string),
			"vpc-id":                 vpcId.(string),
			"association.subnet-id":  subnetId.(string),
			"association.gateway-id": gatewayId.(string),
		},
	)
	req.Filters = append(req.Filters, buildEC2TagFilterList(
		tftags.New(tags.(map[string]interface{})).Ec2Tags(),
	)...)
	req.Filters = append(req.Filters, buildEC2CustomFilterList(
		filter.(*schema.Set),
	)...)

	log.Printf("[DEBUG] Reading Route Table: %s", req)
	resp, err := conn.DescribeRouteTables(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.RouteTables) == 0 {
		return fmt.Errorf("query returned no results. Please change your search criteria and try again")
	}
	if len(resp.RouteTables) > 1 {
		return fmt.Errorf("multiple Route Tables matched; use additional constraints to reduce matches to a single Route Table")
	}

	rt := resp.RouteTables[0]

	d.SetId(aws.StringValue(rt.RouteTableId))

	ownerID := aws.StringValue(rt.OwnerId)
	arn := arn.ARN{
		Partition: meta.(*conns.AWSClient).Partition,
		Service:   ec2.ServiceName,
		Region:    meta.(*conns.AWSClient).Region,
		AccountID: ownerID,
		Resource:  fmt.Sprintf("route-table/%s", d.Id()),
	}.String()
	d.Set("arn", arn)
	d.Set("owner_id", ownerID)

	d.Set("route_table_id", rt.RouteTableId)
	d.Set("vpc_id", rt.VpcId)

	if err := d.Set("tags", tftags.Ec2KeyValueTags(rt.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("routes", dataSourceRoutesRead(rt.Routes)); err != nil {
		return err
	}

	if err := d.Set("associations", dataSourceAssociationsRead(rt.Associations)); err != nil {
		return err
	}

	return nil
}

func dataSourceRoutesRead(ec2Routes []*ec2.Route) []map[string]interface{} {
	routes := make([]map[string]interface{}, 0, len(ec2Routes))
	// Loop through the routes and add them to the set
	for _, r := range ec2Routes {
		if r.GatewayId != nil && *r.GatewayId == "local" {
			continue
		}

		if r.Origin != nil && *r.Origin == "EnableVgwRoutePropagation" {
			continue
		}

		if r.DestinationPrefixListId != nil && strings.HasPrefix(aws.StringValue(r.GatewayId), "vpce-") {
			// Skipping because VPC endpoint routes are handled separately
			// See aws_vpc_endpoint
			continue
		}

		m := make(map[string]interface{})

		if r.DestinationCidrBlock != nil {
			m["cidr_block"] = *r.DestinationCidrBlock
		}
		if r.DestinationIpv6CidrBlock != nil {
			m["ipv6_cidr_block"] = *r.DestinationIpv6CidrBlock
		}
		if r.DestinationPrefixListId != nil {
			m["destination_prefix_list_id"] = *r.DestinationPrefixListId
		}
		if r.CarrierGatewayId != nil {
			m["carrier_gateway_id"] = *r.CarrierGatewayId
		}
		if r.EgressOnlyInternetGatewayId != nil {
			m["egress_only_gateway_id"] = *r.EgressOnlyInternetGatewayId
		}
		if r.GatewayId != nil {
			if strings.HasPrefix(*r.GatewayId, "vpce-") {
				m["vpc_endpoint_id"] = *r.GatewayId
			} else {
				m["gateway_id"] = *r.GatewayId
			}
		}
		if r.NatGatewayId != nil {
			m["nat_gateway_id"] = *r.NatGatewayId
		}
		if r.LocalGatewayId != nil {
			m["local_gateway_id"] = *r.LocalGatewayId
		}
		if r.InstanceId != nil {
			m["instance_id"] = *r.InstanceId
		}
		if r.TransitGatewayId != nil {
			m["transit_gateway_id"] = *r.TransitGatewayId
		}
		if r.VpcPeeringConnectionId != nil {
			m["vpc_peering_connection_id"] = *r.VpcPeeringConnectionId
		}
		if r.NetworkInterfaceId != nil {
			m["network_interface_id"] = *r.NetworkInterfaceId
		}

		routes = append(routes, m)
	}
	return routes
}

func dataSourceAssociationsRead(ec2Assocations []*ec2.RouteTableAssociation) []map[string]interface{} {
	associations := make([]map[string]interface{}, 0, len(ec2Assocations))
	// Loop through the routes and add them to the set
	for _, a := range ec2Assocations {

		m := make(map[string]interface{})
		m["route_table_id"] = *a.RouteTableId
		m["route_table_association_id"] = *a.RouteTableAssociationId
		// GH[11134]
		if a.SubnetId != nil {
			m["subnet_id"] = *a.SubnetId
		}
		if a.GatewayId != nil {
			m["gateway_id"] = *a.GatewayId
		}
		m["main"] = *a.Main
		associations = append(associations, m)
	}
	return associations
}

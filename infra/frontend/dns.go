package frontend

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cloudfront"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// CloudFront's global hosted zone ID for alias records.
const cloudFrontHostedZoneID = "Z2FDTNDATAQYW2"

// createAliasRecords wires the site (and optional www) records to CloudFront.
func createAliasRecords(
	ctx *pulumi.Context,
	zoneID pulumi.StringInput,
	siteFQDN, siteSub, domain string,
	distribution *cloudfront.Distribution,
) error {
	_, err := route53.NewRecord(ctx, "site-a", &route53.RecordArgs{
		ZoneId: zoneID,
		Name:   pulumi.String(siteFQDN),
		Type:   pulumi.String("A"),
		Aliases: route53.RecordAliasArray{
			&route53.RecordAliasArgs{
				Name:                 distribution.DomainName,
				ZoneId:               pulumi.String(cloudFrontHostedZoneID),
				EvaluateTargetHealth: pulumi.Bool(false),
			},
		},
	})
	if err != nil {
		return err
	}
	_, err = route53.NewRecord(ctx, "site-aaaa", &route53.RecordArgs{
		ZoneId: zoneID,
		Name:   pulumi.String(siteFQDN),
		Type:   pulumi.String("AAAA"),
		Aliases: route53.RecordAliasArray{
			&route53.RecordAliasArgs{
				Name:                 distribution.DomainName,
				ZoneId:               pulumi.String(cloudFrontHostedZoneID),
				EvaluateTargetHealth: pulumi.Bool(false),
			},
		},
	})
	if err != nil {
		return err
	}
	if siteSub == "" {
		_, err = route53.NewRecord(ctx, "site-www-a", &route53.RecordArgs{
			ZoneId: zoneID,
			Name:   pulumi.String("www." + domain),
			Type:   pulumi.String("A"),
			Aliases: route53.RecordAliasArray{
				&route53.RecordAliasArgs{
					Name:                 distribution.DomainName,
					ZoneId:               pulumi.String(cloudFrontHostedZoneID),
					EvaluateTargetHealth: pulumi.Bool(false),
				},
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

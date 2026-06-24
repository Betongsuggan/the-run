package main

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigatewayv2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cloudfront"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// CloudFront's global hosted zone ID for alias records.
const cloudFrontHostedZoneID = "Z2FDTNDATAQYW2"

// setupHostedZone either creates the Route53 hosted zone or looks up an existing one.
func setupHostedZone(
	ctx *pulumi.Context,
	domain string,
	createZone bool,
) (pulumi.StringInput, pulumi.StringArrayOutput, error) {
	if createZone {
		zone, err := route53.NewZone(ctx, "zone", &route53.ZoneArgs{
			Name: pulumi.String(domain),
		})
		if err != nil {
			return nil, pulumi.StringArrayOutput{}, err
		}
		return zone.ID(), zone.NameServers, nil
	}
	lookup, err := route53.LookupZone(ctx, &route53.LookupZoneArgs{
		Name: pulumi.StringRef(domain + "."),
	})
	if err != nil {
		return nil, pulumi.StringArrayOutput{}, err
	}
	return pulumi.String(lookup.ZoneId), pulumi.ToStringArray(lookup.NameServers).ToStringArrayOutput(), nil
}

// createCertValidationRecords creates the DNS records ACM needs to validate the cert,
// returning the FQDNs to hand to acm.CertificateValidation. It indexes into
// cert.DomainValidationOptions at compile-time-known positions (one per DomainName + SAN),
// so resources are registered up-front (not inside an ApplyT closure).
func createCertValidationRecords(
	ctx *pulumi.Context,
	name string,
	cert *acm.Certificate,
	zoneID pulumi.StringInput,
	count int,
) (pulumi.StringArrayOutput, error) {
	fqdns := make(pulumi.StringArray, 0, count)
	for i := 0; i < count; i++ {
		opt := cert.DomainValidationOptions.Index(pulumi.Int(i))
		recName := opt.ResourceRecordName().Elem()
		recType := opt.ResourceRecordType().Elem()
		recValue := opt.ResourceRecordValue().Elem()
		_, err := route53.NewRecord(ctx, fmt.Sprintf("%s-validation-%d", name, i), &route53.RecordArgs{
			ZoneId:         zoneID,
			Name:           recName,
			Type:           recType,
			Records:        pulumi.StringArray{recValue},
			Ttl:            pulumi.Int(60),
			AllowOverwrite: pulumi.Bool(true),
		})
		if err != nil {
			return pulumi.StringArrayOutput{}, err
		}
		fqdns = append(fqdns, recName)
	}
	return fqdns.ToStringArrayOutput(), nil
}

// setupAliasRecords wires the site (and optional www) records to CloudFront,
// and the API record to the API Gateway custom domain.
func setupAliasRecords(
	ctx *pulumi.Context,
	zoneID pulumi.StringInput,
	siteFQDN, siteSub, domain, apiFQDN string,
	distribution *cloudfront.Distribution,
	apiDomain *apigatewayv2.DomainName,
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

	apiTargetName := apiDomain.DomainNameConfiguration.TargetDomainName().Elem()
	apiTargetZoneID := apiDomain.DomainNameConfiguration.HostedZoneId().Elem()
	_, err = route53.NewRecord(ctx, "api-a", &route53.RecordArgs{
		ZoneId: zoneID,
		Name:   pulumi.String(apiFQDN),
		Type:   pulumi.String("A"),
		Aliases: route53.RecordAliasArray{
			&route53.RecordAliasArgs{
				Name:                 apiTargetName,
				ZoneId:               apiTargetZoneID,
				EvaluateTargetHealth: pulumi.Bool(false),
			},
		},
	})
	return err
}

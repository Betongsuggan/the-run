// Package dns holds DNS helpers shared by the frontend and backend
// subpackages: hosted-zone lookup/creation and ACM cert validation records.
package dns

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// SetupHostedZone either creates the Route53 hosted zone or looks up an existing one.
func SetupHostedZone(
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

// CreateCertValidationRecords creates the DNS records ACM needs to validate the cert,
// returning the FQDNs to hand to acm.CertificateValidation. It indexes into
// cert.DomainValidationOptions at compile-time-known positions (one per DomainName + SAN),
// so resources are registered up-front (not inside an ApplyT closure).
func CreateCertValidationRecords(
	ctx *pulumi.Context,
	name string,
	cert *acm.Certificate,
	zoneID pulumi.StringInput,
	count int,
) (pulumi.StringArrayOutput, error) {
	fqdns := make(pulumi.StringArray, 0, count)
	for i := range count {
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

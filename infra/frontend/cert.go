package frontend

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/acm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	infradns "github.com/BirgerRydback/the-run/infra/dns"
)

// createCert issues the ACM cert in us-east-1 used by the CloudFront
// distribution. On an apex deploy it also covers www.<domain>.
func createCert(
	ctx *pulumi.Context,
	siteFQDN, siteSub, domain string,
	zoneID pulumi.StringInput,
	usEast1 *aws.Provider,
) (*acm.Certificate, *acm.CertificateValidation, error) {
	cfDomains := []string{siteFQDN}
	var cfSANs pulumi.StringArray
	if siteSub == "" {
		cfDomains = append(cfDomains, "www."+domain)
		cfSANs = pulumi.StringArray{pulumi.String("www." + domain)}
	}
	cert, err := acm.NewCertificate(ctx, "cf-cert", &acm.CertificateArgs{
		DomainName:              pulumi.String(siteFQDN),
		SubjectAlternativeNames: cfSANs,
		ValidationMethod:        pulumi.String("DNS"),
	}, pulumi.Provider(usEast1))
	if err != nil {
		return nil, nil, err
	}
	validationFQDNs, err := infradns.CreateCertValidationRecords(ctx, "cf-cert", cert, zoneID, len(cfDomains))
	if err != nil {
		return nil, nil, err
	}
	validation, err := acm.NewCertificateValidation(ctx, "cf-cert-validation", &acm.CertificateValidationArgs{
		CertificateArn:        cert.Arn,
		ValidationRecordFqdns: validationFQDNs,
	}, pulumi.Provider(usEast1))
	if err != nil {
		return nil, nil, err
	}
	return cert, validation, nil
}

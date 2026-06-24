package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/acm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// createCloudFrontCert issues the ACM cert in us-east-1 used by the CloudFront
// distribution. On an apex deploy it also covers www.<domain>.
func createCloudFrontCert(
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
	cfCert, err := acm.NewCertificate(ctx, "cf-cert", &acm.CertificateArgs{
		DomainName:              pulumi.String(siteFQDN),
		SubjectAlternativeNames: cfSANs,
		ValidationMethod:        pulumi.String("DNS"),
	}, pulumi.Provider(usEast1))
	if err != nil {
		return nil, nil, err
	}
	cfValidationFQDNs, err := createCertValidationRecords(ctx, "cf-cert", cfCert, zoneID, len(cfDomains))
	if err != nil {
		return nil, nil, err
	}
	cfCertValidation, err := acm.NewCertificateValidation(ctx, "cf-cert-validation", &acm.CertificateValidationArgs{
		CertificateArn:        cfCert.Arn,
		ValidationRecordFqdns: cfValidationFQDNs,
	}, pulumi.Provider(usEast1))
	if err != nil {
		return nil, nil, err
	}
	return cfCert, cfCertValidation, nil
}

// createAPICert issues the regional ACM cert used by the API Gateway custom domain.
func createAPICert(
	ctx *pulumi.Context,
	apiFQDN string,
	zoneID pulumi.StringInput,
) (*acm.Certificate, *acm.CertificateValidation, error) {
	apiCert, err := acm.NewCertificate(ctx, "api-cert", &acm.CertificateArgs{
		DomainName:       pulumi.String(apiFQDN),
		ValidationMethod: pulumi.String("DNS"),
	})
	if err != nil {
		return nil, nil, err
	}
	apiValidationFQDNs, err := createCertValidationRecords(ctx, "api-cert", apiCert, zoneID, 1)
	if err != nil {
		return nil, nil, err
	}
	apiCertValidation, err := acm.NewCertificateValidation(
		ctx,
		"api-cert-validation",
		&acm.CertificateValidationArgs{
			CertificateArn:        apiCert.Arn,
			ValidationRecordFqdns: apiValidationFQDNs,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	return apiCert, apiCertValidation, nil
}

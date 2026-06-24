package backend

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/acm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	infradns "github.com/BirgerRydback/the-run/infra/dns"
)

// createCert issues the regional ACM cert used by the API Gateway custom domain.
func createCert(
	ctx *pulumi.Context,
	apiFQDN string,
	zoneID pulumi.StringInput,
) (*acm.Certificate, *acm.CertificateValidation, error) {
	cert, err := acm.NewCertificate(ctx, "api-cert", &acm.CertificateArgs{
		DomainName:       pulumi.String(apiFQDN),
		ValidationMethod: pulumi.String("DNS"),
	})
	if err != nil {
		return nil, nil, err
	}
	validationFQDNs, err := infradns.CreateCertValidationRecords(ctx, "api-cert", cert, zoneID, 1)
	if err != nil {
		return nil, nil, err
	}
	validation, err := acm.NewCertificateValidation(
		ctx,
		"api-cert-validation",
		&acm.CertificateValidationArgs{
			CertificateArn:        cert.Arn,
			ValidationRecordFqdns: validationFQDNs,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	return cert, validation, nil
}

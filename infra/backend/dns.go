package backend

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigatewayv2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// createAliasRecord wires the api FQDN to the API Gateway custom domain.
func createAliasRecord(
	ctx *pulumi.Context,
	zoneID pulumi.StringInput,
	apiFQDN string,
	apiDomain *apigatewayv2.DomainName,
) error {
	apiTargetName := apiDomain.DomainNameConfiguration.TargetDomainName().Elem()
	apiTargetZoneID := apiDomain.DomainNameConfiguration.HostedZoneId().Elem()
	_, err := route53.NewRecord(ctx, "api-a", &route53.RecordArgs{
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

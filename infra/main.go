package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "the-run")
		domain := cfg.Require("domain")
		apiSub := cfg.Get("apiSubdomain")
		if apiSub == "" {
			apiSub = "api"
		}
		siteSub := cfg.Get("siteSubdomain") // "" => apex
		createZone := cfg.GetBool("createHostedZone")

		apiFQDN := apiSub + "." + domain
		siteFQDN := domain
		if siteSub != "" {
			siteFQDN = siteSub + "." + domain
		}

		// us-east-1 provider for the CloudFront ACM certificate.
		usEast1, err := aws.NewProvider(ctx, "us-east-1", &aws.ProviderArgs{
			Region: pulumi.String("us-east-1"),
		})
		if err != nil {
			return err
		}

		zoneID, nameServers, err := setupHostedZone(ctx, domain, createZone)
		if err != nil {
			return err
		}

		cfCert, cfCertValidation, err := createCloudFrontCert(ctx, siteFQDN, siteSub, domain, zoneID, usEast1)
		if err != nil {
			return err
		}
		apiCert, apiCertValidation, err := createAPICert(ctx, apiFQDN, zoneID)
		if err != nil {
			return err
		}

		fe, err := setupFrontend(ctx, siteSub, domain, siteFQDN, cfCert, cfCertValidation)
		if err != nil {
			return err
		}

		ap, err := setupAPI(ctx, apiFQDN, siteFQDN, apiCert, apiCertValidation)
		if err != nil {
			return err
		}

		if err := setupAliasRecords(ctx, zoneID, siteFQDN, siteSub, domain, apiFQDN, fe.distribution, ap.apiDomain); err != nil {
			return err
		}

		ctx.Export("frontendUrl", pulumi.Sprintf("https://%s", siteFQDN))
		ctx.Export("apiUrl", pulumi.Sprintf("https://%s", apiFQDN))
		ctx.Export("apiGatewayInvokeUrl", ap.api.ApiEndpoint)
		ctx.Export("cloudFrontDistributionId", fe.distribution.ID())
		ctx.Export("cloudFrontDomainName", fe.distribution.DomainName)
		ctx.Export("frontendBucket", fe.bucket.Bucket)
		ctx.Export("lambdaName", ap.function.Name)
		ctx.Export("hostedZoneNameServers", nameServers)

		return nil
	})
}

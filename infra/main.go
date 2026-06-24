package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"github.com/BirgerRydback/the-run/infra/backend"
	infradns "github.com/BirgerRydback/the-run/infra/dns"
	"github.com/BirgerRydback/the-run/infra/frontend"
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

		usEast1, err := aws.NewProvider(ctx, "us-east-1", &aws.ProviderArgs{
			Region: pulumi.String("us-east-1"),
		})
		if err != nil {
			return err
		}

		zoneID, nameServers, err := infradns.SetupHostedZone(ctx, domain, createZone)
		if err != nil {
			return err
		}

		fe, err := frontend.Setup(ctx, siteSub, domain, siteFQDN, zoneID, usEast1)
		if err != nil {
			return err
		}

		be, err := backend.Setup(ctx, apiFQDN, siteFQDN, zoneID)
		if err != nil {
			return err
		}

		ctx.Export("frontendUrl", pulumi.Sprintf("https://%s", siteFQDN))
		ctx.Export("apiUrl", pulumi.Sprintf("https://%s", apiFQDN))
		ctx.Export("apiGatewayInvokeUrl", be.API.ApiEndpoint)
		ctx.Export("cloudFrontDistributionId", fe.Distribution.ID())
		ctx.Export("cloudFrontDomainName", fe.Distribution.DomainName)
		ctx.Export("frontendBucket", fe.Bucket.Bucket)
		ctx.Export("lambdaName", be.Function.Name)
		ctx.Export("hostedZoneNameServers", nameServers)

		return nil
	})
}

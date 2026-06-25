package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"github.com/BirgerRydback/the-run/infra/backend"
	"github.com/BirgerRydback/the-run/infra/database"
	infradns "github.com/BirgerRydback/the-run/infra/dns"
	"github.com/BirgerRydback/the-run/infra/email"
	"github.com/BirgerRydback/the-run/infra/frontend"
)

const localstackEndpoint = "http://localhost:4566"

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "the-run")

		if cfg.GetBool("localstack") {
			return runLocalstack(ctx)
		}

		return runAWS(ctx, cfg)
	})
}

// runLocalstack provisions only the DynamoDB tables, pointed at a local
// LocalStack endpoint. The Go backend runs natively against these tables in
// development — we don't deploy the Lambda/API Gateway/CloudFront/Route53
// resources into LocalStack (they need LocalStack Pro and we have no need for
// them locally).
func runLocalstack(ctx *pulumi.Context) error {
	// Endpoint routing comes from AWS_ENDPOINT_URL in the environment (set
	// by the `localstack-*` Justfile recipes). The AWS SDK v2 treats that
	// env var as a global override for every service, so we don't need to
	// enumerate per-service endpoints here — and we avoid the trap where
	// listing only DynamoDB would send tagging/STS calls to real AWS and
	// hang.
	provider, err := aws.NewProvider(ctx, "localstack", &aws.ProviderArgs{
		Region:                    pulumi.String("eu-north-1"),
		AccessKey:                 pulumi.String("test"),
		SecretKey:                 pulumi.String("test"),
		SkipCredentialsValidation: pulumi.Bool(true),
		SkipRequestingAccountId:   pulumi.Bool(true),
		SkipMetadataApiCheck:      pulumi.Bool(true),
		S3UsePathStyle:            pulumi.Bool(true),
	})
	if err != nil {
		return err
	}

	tables, err := database.Setup(ctx, provider)
	if err != nil {
		return err
	}

	ctx.Export("runnersTableName", tables.Runners.Name)
	ctx.Export("registrationsTableName", tables.Registrations.Name)
	ctx.Export("eventsTableName", tables.Events.Name)
	ctx.Export("racesTableName", tables.Races.Name)
	ctx.Export("accountsTableName", tables.Accounts.Name)
	ctx.Export("authAttemptsTableName", tables.AuthAttempts.Name)
	ctx.Export("localstackEndpoint", pulumi.String(localstackEndpoint))
	return nil
}

func runAWS(ctx *pulumi.Context, cfg *config.Config) error {
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

	tables, err := database.Setup(ctx, nil)
	if err != nil {
		return err
	}

	// SES sending domain + DKIM + MAIL FROM + configuration set. Senders
	// (guardian magic links, DSR magic links) plug in via backend env vars
	// — see infra/backend below.
	em, err := email.Setup(ctx, siteFQDN, "eu-north-1", zoneID)
	if err != nil {
		return err
	}

	be, err := backend.Setup(ctx, apiFQDN, siteFQDN, domain, zoneID, tables, em)
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
	ctx.Export("runnersTableName", tables.Runners.Name)
	ctx.Export("registrationsTableName", tables.Registrations.Name)
	ctx.Export("eventsTableName", tables.Events.Name)
	ctx.Export("racesTableName", tables.Races.Name)
	ctx.Export("accountsTableName", tables.Accounts.Name)
	ctx.Export("authAttemptsTableName", tables.AuthAttempts.Name)
	ctx.Export("jwtSecretArn", be.JWTSecret.Arn)
	ctx.Export("hostedZoneNameServers", nameServers)
	ctx.Export("sesSendingDomain", em.Identity.EmailIdentity)
	ctx.Export("sesMailFromDomain", pulumi.String(em.MailFromDomain))
	ctx.Export("sesConfigurationSet", em.ConfigurationSet.ConfigurationSetName)
	ctx.Export("sesSenderAddress", pulumi.String(em.SenderAddress))

	return nil
}

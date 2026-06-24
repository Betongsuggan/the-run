package main

import (
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigatewayv2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cloudfront"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	awslambda "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/s3"
	syncedfolder "github.com/pulumi/pulumi-synced-folder/sdk/go/synced-folder"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// CloudFront's global hosted zone ID for alias records.
const cloudFrontHostedZoneID = "Z2FDTNDATAQYW2"

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

		// 1. Route53 hosted zone.
		var zoneID pulumi.StringInput
		var nameServers pulumi.StringArrayOutput
		if createZone {
			zone, err := route53.NewZone(ctx, "zone", &route53.ZoneArgs{
				Name: pulumi.String(domain),
			})
			if err != nil {
				return err
			}
			zoneID = zone.ID()
			nameServers = zone.NameServers
		} else {
			lookup, err := route53.LookupZone(ctx, &route53.LookupZoneArgs{
				Name: pulumi.StringRef(domain + "."),
			})
			if err != nil {
				return err
			}
			zoneID = pulumi.String(lookup.ZoneId)
			nameServers = pulumi.ToStringArray(lookup.NameServers).ToStringArrayOutput()
		}

		// 2. CloudFront ACM cert (us-east-1). Covers site FQDN, plus www when on apex.
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
			return err
		}
		cfValidationFQDNs, err := createCertValidationRecords(ctx, "cf-cert", cfCert, zoneID, len(cfDomains))
		if err != nil {
			return err
		}
		cfCertValidation, err := acm.NewCertificateValidation(ctx, "cf-cert-validation", &acm.CertificateValidationArgs{
			CertificateArn:        cfCert.Arn,
			ValidationRecordFqdns: cfValidationFQDNs,
		}, pulumi.Provider(usEast1))
		if err != nil {
			return err
		}

		// 3. API Gateway ACM cert (default region = eu-north-1).
		apiCert, err := acm.NewCertificate(ctx, "api-cert", &acm.CertificateArgs{
			DomainName:       pulumi.String(apiFQDN),
			ValidationMethod: pulumi.String("DNS"),
		})
		if err != nil {
			return err
		}
		apiValidationFQDNs, err := createCertValidationRecords(ctx, "api-cert", apiCert, zoneID, 1)
		if err != nil {
			return err
		}
		apiCertValidation, err := acm.NewCertificateValidation(ctx, "api-cert-validation", &acm.CertificateValidationArgs{
			CertificateArn:        apiCert.Arn,
			ValidationRecordFqdns: apiValidationFQDNs,
		})
		if err != nil {
			return err
		}

		// 4. Frontend S3 bucket (private, OAC-only).
		siteBucket, err := s3.NewBucketV2(ctx, "site-bucket", &s3.BucketV2Args{
			ForceDestroy: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		_, err = s3.NewBucketPublicAccessBlock(ctx, "site-bucket-bpa", &s3.BucketPublicAccessBlockArgs{
			Bucket:                siteBucket.ID(),
			BlockPublicAcls:       pulumi.Bool(true),
			BlockPublicPolicy:     pulumi.Bool(true),
			IgnorePublicAcls:      pulumi.Bool(true),
			RestrictPublicBuckets: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		_, err = s3.NewBucketServerSideEncryptionConfigurationV2(ctx, "site-bucket-sse", &s3.BucketServerSideEncryptionConfigurationV2Args{
			Bucket: siteBucket.ID(),
			Rules: s3.BucketServerSideEncryptionConfigurationV2RuleArray{
				&s3.BucketServerSideEncryptionConfigurationV2RuleArgs{
					ApplyServerSideEncryptionByDefault: &s3.BucketServerSideEncryptionConfigurationV2RuleApplyServerSideEncryptionByDefaultArgs{
						SseAlgorithm: pulumi.String("AES256"),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		// 5. CloudFront OAC.
		oac, err := cloudfront.NewOriginAccessControl(ctx, "site-oac", &cloudfront.OriginAccessControlArgs{
			Description:                   pulumi.String("OAC for the-run site bucket"),
			OriginAccessControlOriginType: pulumi.String("s3"),
			SigningBehavior:               pulumi.String("always"),
			SigningProtocol:               pulumi.String("sigv4"),
		})
		if err != nil {
			return err
		}

		// 6. CloudFront distribution.
		aliases := pulumi.StringArray{pulumi.String(siteFQDN)}
		if siteSub == "" {
			aliases = append(aliases, pulumi.String("www."+domain))
		}
		distribution, err := cloudfront.NewDistribution(ctx, "site-distribution", &cloudfront.DistributionArgs{
			Enabled:           pulumi.Bool(true),
			IsIpv6Enabled:     pulumi.Bool(true),
			Comment:           pulumi.String("the-run frontend"),
			DefaultRootObject: pulumi.String("index.html"),
			Aliases:           aliases,
			Origins: cloudfront.DistributionOriginArray{
				&cloudfront.DistributionOriginArgs{
					OriginId:              siteBucket.ID(),
					DomainName:            siteBucket.BucketRegionalDomainName,
					OriginAccessControlId: oac.ID(),
				},
			},
			DefaultCacheBehavior: &cloudfront.DistributionDefaultCacheBehaviorArgs{
				TargetOriginId:       siteBucket.ID(),
				ViewerProtocolPolicy: pulumi.String("redirect-to-https"),
				AllowedMethods:       pulumi.StringArray{pulumi.String("GET"), pulumi.String("HEAD"), pulumi.String("OPTIONS")},
				CachedMethods:        pulumi.StringArray{pulumi.String("GET"), pulumi.String("HEAD")},
				Compress:             pulumi.Bool(true),
				// AWS-managed CachingOptimized policy.
				CachePolicyId: pulumi.String("658327ea-f89d-4fab-a63d-7e88639e58f6"),
			},
			CustomErrorResponses: cloudfront.DistributionCustomErrorResponseArray{
				&cloudfront.DistributionCustomErrorResponseArgs{
					ErrorCode:        pulumi.Int(403),
					ResponseCode:     pulumi.Int(200),
					ResponsePagePath: pulumi.String("/index.html"),
				},
				&cloudfront.DistributionCustomErrorResponseArgs{
					ErrorCode:        pulumi.Int(404),
					ResponseCode:     pulumi.Int(200),
					ResponsePagePath: pulumi.String("/index.html"),
				},
			},
			Restrictions: &cloudfront.DistributionRestrictionsArgs{
				GeoRestriction: &cloudfront.DistributionRestrictionsGeoRestrictionArgs{
					RestrictionType: pulumi.String("none"),
				},
			},
			ViewerCertificate: &cloudfront.DistributionViewerCertificateArgs{
				AcmCertificateArn:      cfCert.Arn,
				SslSupportMethod:       pulumi.String("sni-only"),
				MinimumProtocolVersion: pulumi.String("TLSv1.2_2021"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{cfCertValidation}))
		if err != nil {
			return err
		}

		// Bucket policy: only CloudFront-via-OAC may read.
		bucketPolicyDoc := pulumi.All(siteBucket.Arn, distribution.Arn).ApplyT(func(args []any) (string, error) {
			bucketArn := args[0].(string)
			distArn := args[1].(string)
			doc := map[string]any{
				"Version": "2012-10-17",
				"Statement": []any{
					map[string]any{
						"Sid":       "AllowCloudFrontServicePrincipal",
						"Effect":    "Allow",
						"Principal": map[string]any{"Service": "cloudfront.amazonaws.com"},
						"Action":    "s3:GetObject",
						"Resource":  bucketArn + "/*",
						"Condition": map[string]any{
							"StringEquals": map[string]any{"AWS:SourceArn": distArn},
						},
					},
				},
			}
			b, err := json.Marshal(doc)
			if err != nil {
				return "", err
			}
			return string(b), nil
		}).(pulumi.StringOutput)
		_, err = s3.NewBucketPolicy(ctx, "site-bucket-policy", &s3.BucketPolicyArgs{
			Bucket: siteBucket.ID(),
			Policy: bucketPolicyDoc,
		})
		if err != nil {
			return err
		}

		// 7. Sync built frontend assets into the bucket.
		_, err = syncedfolder.NewS3BucketFolder(ctx, "site-folder", &syncedfolder.S3BucketFolderArgs{
			Path:       pulumi.String("../frontend/build"),
			BucketName: siteBucket.Bucket,
			Acl:        pulumi.String("private"),
		})
		if err != nil {
			return err
		}

		// 8. Lambda IAM role.
		assumePolicy, err := json.Marshal(map[string]any{
			"Version": "2012-10-17",
			"Statement": []any{
				map[string]any{
					"Effect":    "Allow",
					"Principal": map[string]any{"Service": "lambda.amazonaws.com"},
					"Action":    "sts:AssumeRole",
				},
			},
		})
		if err != nil {
			return err
		}
		lambdaRole, err := iam.NewRole(ctx, "lambda-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(string(assumePolicy)),
		})
		if err != nil {
			return err
		}
		_, err = iam.NewRolePolicyAttachment(ctx, "lambda-basic-exec", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
		})
		if err != nil {
			return err
		}

		// 9. Lambda. Expects ../backend/dist/lambda.zip prebuilt.
		fn, err := awslambda.NewFunction(ctx, "api-fn", &awslambda.FunctionArgs{
			Role:          lambdaRole.Arn,
			Runtime:       pulumi.String("provided.al2023"),
			Architectures: pulumi.StringArray{pulumi.String("arm64")},
			Handler:       pulumi.String("bootstrap"),
			Code:          pulumi.NewFileArchive("../backend/dist/lambda.zip"),
			MemorySize:    pulumi.Int(256),
			Timeout:       pulumi.Int(10),
		})
		if err != nil {
			return err
		}

		// 10. API Gateway v2 HTTP API.
		api, err := apigatewayv2.NewApi(ctx, "api", &apigatewayv2.ApiArgs{
			ProtocolType: pulumi.String("HTTP"),
			CorsConfiguration: &apigatewayv2.ApiCorsConfigurationArgs{
				AllowOrigins: pulumi.StringArray{pulumi.String("https://" + siteFQDN)},
				AllowMethods: pulumi.StringArray{pulumi.String("GET"), pulumi.String("POST"), pulumi.String("PUT"), pulumi.String("DELETE"), pulumi.String("OPTIONS")},
				AllowHeaders: pulumi.StringArray{pulumi.String("content-type"), pulumi.String("authorization")},
				MaxAge:       pulumi.Int(300),
			},
		})
		if err != nil {
			return err
		}
		integration, err := apigatewayv2.NewIntegration(ctx, "api-integration", &apigatewayv2.IntegrationArgs{
			ApiId:                api.ID(),
			IntegrationType:      pulumi.String("AWS_PROXY"),
			IntegrationUri:       fn.InvokeArn,
			IntegrationMethod:    pulumi.String("POST"),
			PayloadFormatVersion: pulumi.String("2.0"),
		})
		if err != nil {
			return err
		}
		_, err = apigatewayv2.NewRoute(ctx, "api-route-default", &apigatewayv2.RouteArgs{
			ApiId:    api.ID(),
			RouteKey: pulumi.String("ANY /{proxy+}"),
			Target:   pulumi.Sprintf("integrations/%s", integration.ID()),
		})
		if err != nil {
			return err
		}
		_, err = apigatewayv2.NewRoute(ctx, "api-route-root", &apigatewayv2.RouteArgs{
			ApiId:    api.ID(),
			RouteKey: pulumi.String("ANY /"),
			Target:   pulumi.Sprintf("integrations/%s", integration.ID()),
		})
		if err != nil {
			return err
		}
		stage, err := apigatewayv2.NewStage(ctx, "api-stage", &apigatewayv2.StageArgs{
			ApiId:      api.ID(),
			Name:       pulumi.String("$default"),
			AutoDeploy: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		_, err = awslambda.NewPermission(ctx, "api-invoke-lambda", &awslambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  fn.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("%s/*/*", api.ExecutionArn),
		})
		if err != nil {
			return err
		}

		// 11. Custom API domain + mapping.
		apiDomain, err := apigatewayv2.NewDomainName(ctx, "api-domain", &apigatewayv2.DomainNameArgs{
			DomainName: pulumi.String(apiFQDN),
			DomainNameConfiguration: &apigatewayv2.DomainNameDomainNameConfigurationArgs{
				CertificateArn: apiCert.Arn,
				EndpointType:   pulumi.String("REGIONAL"),
				SecurityPolicy: pulumi.String("TLS_1_2"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{apiCertValidation}))
		if err != nil {
			return err
		}
		_, err = apigatewayv2.NewApiMapping(ctx, "api-mapping", &apigatewayv2.ApiMappingArgs{
			ApiId:      api.ID(),
			DomainName: apiDomain.ID(),
			Stage:      stage.Name,
		})
		if err != nil {
			return err
		}

		// 12. Route53 alias records.
		_, err = route53.NewRecord(ctx, "site-a", &route53.RecordArgs{
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

		apiTargetName := apiDomain.DomainNameConfiguration.ApplyT(func(c *apigatewayv2.DomainNameDomainNameConfiguration) string {
			if c == nil || c.TargetDomainName == nil {
				return ""
			}
			return *c.TargetDomainName
		}).(pulumi.StringOutput)
		apiTargetZoneID := apiDomain.DomainNameConfiguration.ApplyT(func(c *apigatewayv2.DomainNameDomainNameConfiguration) string {
			if c == nil || c.HostedZoneId == nil {
				return ""
			}
			return *c.HostedZoneId
		}).(pulumi.StringOutput)
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
		if err != nil {
			return err
		}

		// 13. Stack outputs.
		ctx.Export("frontendUrl", pulumi.Sprintf("https://%s", siteFQDN))
		ctx.Export("apiUrl", pulumi.Sprintf("https://%s", apiFQDN))
		ctx.Export("apiGatewayInvokeUrl", api.ApiEndpoint)
		ctx.Export("cloudFrontDistributionId", distribution.ID())
		ctx.Export("cloudFrontDomainName", distribution.DomainName)
		ctx.Export("frontendBucket", siteBucket.Bucket)
		ctx.Export("lambdaName", fn.Name)
		ctx.Export("hostedZoneNameServers", nameServers)

		return nil
	})
}

// createCertValidationRecords creates the DNS records ACM needs to validate the cert,
// returning the FQDNs to hand to acm.CertificateValidation. It indexes into
// cert.DomainValidationOptions at compile-time-known positions (one per DomainName + SAN),
// so resources are registered up-front (not inside an ApplyT closure).
func createCertValidationRecords(ctx *pulumi.Context, name string, cert *acm.Certificate, zoneID pulumi.StringInput, count int) (pulumi.StringArrayOutput, error) {
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

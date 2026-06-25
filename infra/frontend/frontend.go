// Package frontend provisions the S3 origin bucket, CloudFront distribution
// (OAC-locked to the bucket), the bucket policy granting only that distribution
// read access, the CloudFront ACM cert (us-east-1), and the Route53 alias
// records pointing at the distribution.
package frontend

import (
	"encoding/json"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cloudfront"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/s3"
	syncedfolder "github.com/pulumi/pulumi-synced-folder/sdk/go/synced-folder"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Resources is the set of frontend resources main.go exports outputs for.
type Resources struct {
	Bucket       *s3.BucketV2
	Distribution *cloudfront.Distribution
}

// Setup provisions the full frontend stack: cert, bucket, distribution,
// bucket policy, synced assets, and alias records.
func Setup(
	ctx *pulumi.Context,
	siteSub, domain, siteFQDN string,
	zoneID pulumi.StringInput,
	usEast1 *aws.Provider,
) (*Resources, error) {
	cert, certValidation, err := createCert(ctx, siteFQDN, siteSub, domain, zoneID, usEast1)
	if err != nil {
		return nil, err
	}

	siteBucket, err := s3.NewBucketV2(ctx, "site-bucket", &s3.BucketV2Args{
		ForceDestroy: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	_, err = s3.NewBucketPublicAccessBlock(ctx, "site-bucket-bpa", &s3.BucketPublicAccessBlockArgs{
		Bucket:                siteBucket.ID(),
		BlockPublicAcls:       pulumi.Bool(true),
		BlockPublicPolicy:     pulumi.Bool(true),
		IgnorePublicAcls:      pulumi.Bool(true),
		RestrictPublicBuckets: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	_, err = s3.NewBucketServerSideEncryptionConfigurationV2(
		ctx,
		"site-bucket-sse",
		&s3.BucketServerSideEncryptionConfigurationV2Args{
			Bucket: siteBucket.ID(),
			Rules: s3.BucketServerSideEncryptionConfigurationV2RuleArray{
				&s3.BucketServerSideEncryptionConfigurationV2RuleArgs{
					ApplyServerSideEncryptionByDefault: &s3.BucketServerSideEncryptionConfigurationV2RuleApplyServerSideEncryptionByDefaultArgs{
						SseAlgorithm: pulumi.String("AES256"),
					},
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	oac, err := cloudfront.NewOriginAccessControl(ctx, "site-oac", &cloudfront.OriginAccessControlArgs{
		Description:                   pulumi.String("OAC for the-run site bucket"),
		OriginAccessControlOriginType: pulumi.String("s3"),
		SigningBehavior:               pulumi.String("always"),
		SigningProtocol:               pulumi.String("sigv4"),
	})
	if err != nil {
		return nil, err
	}

	// Response headers policy (GDPR A1.4) — transport-level hardening that
	// CloudFront adds to every response. The document-level CSP comes from
	// SvelteKit's csp config (rendered as a <meta http-equiv> with proper
	// SHA hashes for the inline theme-init script). We keep CSP off this
	// policy because the meta-tag flavour is what handles hashes; mixing
	// would just confuse browsers about which one wins.
	headersPolicy, err := cloudfront.NewResponseHeadersPolicy(ctx, "site-security-headers", &cloudfront.ResponseHeadersPolicyArgs{
		Comment: pulumi.String("the-run security headers: HSTS, anti-clickjacking, anti-MIME-sniff, referrer + permissions"),
		SecurityHeadersConfig: &cloudfront.ResponseHeadersPolicySecurityHeadersConfigArgs{
			StrictTransportSecurity: &cloudfront.ResponseHeadersPolicySecurityHeadersConfigStrictTransportSecurityArgs{
				// One-year max-age + includeSubDomains is the standard
				// "ready for HSTS preload list" configuration. We don't
				// set `preload` until the site has been stable enough to
				// commit irreversibly — easy to add later.
				AccessControlMaxAgeSec: pulumi.Int(31536000),
				IncludeSubdomains:      pulumi.Bool(true),
				Preload:                pulumi.Bool(false),
				Override:               pulumi.Bool(true),
			},
			ContentTypeOptions: &cloudfront.ResponseHeadersPolicySecurityHeadersConfigContentTypeOptionsArgs{
				Override: pulumi.Bool(true),
			},
			// X-Frame-Options DENY blocks all iframe embedding. CSP
			// frame-ancestors would be stronger but can't be set via
			// meta http-equiv (browser limitation), so this is the
			// belt-and-braces headers-level equivalent.
			FrameOptions: &cloudfront.ResponseHeadersPolicySecurityHeadersConfigFrameOptionsArgs{
				FrameOption: pulumi.String("DENY"),
				Override:    pulumi.Bool(true),
			},
			ReferrerPolicy: &cloudfront.ResponseHeadersPolicySecurityHeadersConfigReferrerPolicyArgs{
				ReferrerPolicy: pulumi.String("strict-origin-when-cross-origin"),
				Override:       pulumi.Bool(true),
			},
		},
		CustomHeadersConfig: &cloudfront.ResponseHeadersPolicyCustomHeadersConfigArgs{
			Items: cloudfront.ResponseHeadersPolicyCustomHeadersConfigItemArray{
				// Permissions-Policy isn't a SecurityHeadersConfig field
				// yet in the AWS provider — set via custom headers. List
				// is "deny everything we don't actively use". Add a
				// feature here when we start needing it.
				&cloudfront.ResponseHeadersPolicyCustomHeadersConfigItemArgs{
					Header:   pulumi.String("Permissions-Policy"),
					Value:    pulumi.String("accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=(), interest-cohort=()"),
					Override: pulumi.Bool(true),
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

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
			AllowedMethods: pulumi.StringArray{
				pulumi.String("GET"),
				pulumi.String("HEAD"),
				pulumi.String("OPTIONS"),
			},
			CachedMethods: pulumi.StringArray{pulumi.String("GET"), pulumi.String("HEAD")},
			Compress:      pulumi.Bool(true),
			// AWS-managed CachingOptimized policy.
			CachePolicyId:           pulumi.String("658327ea-f89d-4fab-a63d-7e88639e58f6"),
			ResponseHeadersPolicyId: headersPolicy.ID(),
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
			AcmCertificateArn:      cert.Arn,
			SslSupportMethod:       pulumi.String("sni-only"),
			MinimumProtocolVersion: pulumi.String("TLSv1.2_2021"),
		},
	}, pulumi.DependsOn([]pulumi.Resource{certValidation}))
	if err != nil {
		return nil, err
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
		return nil, err
	}

	_, err = syncedfolder.NewS3BucketFolder(ctx, "site-folder", &syncedfolder.S3BucketFolderArgs{
		Path:       pulumi.String("../frontend/build"),
		BucketName: siteBucket.Bucket,
		Acl:        pulumi.String("private"),
	})
	if err != nil {
		return nil, err
	}

	if err := createAliasRecords(ctx, zoneID, siteFQDN, siteSub, domain, distribution); err != nil {
		return nil, err
	}

	return &Resources{
		Bucket:       siteBucket,
		Distribution: distribution,
	}, nil
}

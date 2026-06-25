// Package email provisions the AWS SES side of the deployment (GDPR A1.5):
// a verified sending domain with Easy DKIM, a MAIL FROM subdomain for bounce
// handling, a configuration set, and CloudWatch event publishing for
// bounce/complaint visibility.
//
// All resources live in the same region as the rest of the stack (eu-north-1).
// The SES account stays in the sandbox until production access is requested
// out-of-band via AWS Support; sandbox sends only to verified recipients,
// which is enough for end-to-end validation of the pipeline.
//
// This slice ships the infrastructure only. Senders for guardian magic links
// (A0.4) and DSR magic links (A1.1) plug in via the Go email package and use
// the ConfigurationSetName + SenderAddress exported here.
package email

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/sesv2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Resources is the public surface — what main.go threads into the backend
// (env vars + IAM grants) and exports as stack outputs.
type Resources struct {
	Identity            *sesv2.EmailIdentity
	ConfigurationSet    *sesv2.ConfigurationSet
	SenderAddress       string // e.g. noreply@running.rydback.net
	MailFromDomain      string // e.g. bounce.running.rydback.net
	ConfigurationSetArn pulumi.StringOutput
}

// Setup wires up SES for the given sending domain (e.g. "running.rydback.net").
// The zoneID is the apex Route53 zone (e.g. for "rydback.net") — all DNS
// records (DKIM CNAMEs, MAIL FROM MX/SPF, DMARC) get published under it.
//
// region is the SES region (e.g. "eu-north-1"); used to build the regional
// MAIL FROM MX target.
func Setup(
	ctx *pulumi.Context,
	sendingDomain, region string,
	zoneID pulumi.StringInput,
) (*Resources, error) {
	mailFromDomain := "bounce." + sendingDomain
	senderAddress := "noreply@" + sendingDomain

	// Easy DKIM domain identity. SES generates 3 DKIM tokens; we publish
	// the matching CNAMEs below. Verification is asynchronous — SES polls
	// DNS and flips VerifiedForSendingStatus when it sees the records.
	identity, err := sesv2.NewEmailIdentity(ctx, "sending-domain", &sesv2.EmailIdentityArgs{
		EmailIdentity: pulumi.String(sendingDomain),
	})
	if err != nil {
		return nil, err
	}

	// Three DKIM CNAMEs. Easy DKIM always yields exactly 3 tokens; indexing
	// is safe and gives Pulumi compile-time-known resource URNs.
	tokens := identity.DkimSigningAttributes.Tokens()
	for i := range 3 {
		token := tokens.Index(pulumi.Int(i))
		_, err := route53.NewRecord(ctx, fmt.Sprintf("dkim-%d", i), &route53.RecordArgs{
			ZoneId:         zoneID,
			Name:           pulumi.Sprintf("%s._domainkey.%s", token, sendingDomain),
			Type:           pulumi.String("CNAME"),
			Ttl:            pulumi.Int(300),
			Records:        pulumi.StringArray{pulumi.Sprintf("%s.dkim.amazonses.com", token)},
			AllowOverwrite: pulumi.Bool(true),
		})
		if err != nil {
			return nil, err
		}
	}

	// MAIL FROM domain: a dedicated subdomain so bounce messages don't muddy
	// the From-domain's reputation. SES requires the MX + SPF below to be
	// resolvable before it'll accept this attribute.
	mailFromMX, err := route53.NewRecord(ctx, "mail-from-mx", &route53.RecordArgs{
		ZoneId:         zoneID,
		Name:           pulumi.String(mailFromDomain),
		Type:           pulumi.String("MX"),
		Ttl:            pulumi.Int(300),
		Records:        pulumi.StringArray{pulumi.String("10 feedback-smtp." + region + ".amazonses.com")},
		AllowOverwrite: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	// TXT record values: the AWS provider quotes them automatically for
	// Route 53, so we pass the bare string. Wrapping it in our own quotes
	// produces "" "v=spf1 ..." "" which Route 53 rejects.
	mailFromSPF, err := route53.NewRecord(ctx, "mail-from-spf", &route53.RecordArgs{
		ZoneId:         zoneID,
		Name:           pulumi.String(mailFromDomain),
		Type:           pulumi.String("TXT"),
		Ttl:            pulumi.Int(300),
		Records:        pulumi.StringArray{pulumi.String("v=spf1 include:amazonses.com ~all")},
		AllowOverwrite: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	_, err = sesv2.NewEmailIdentityMailFromAttributes(ctx, "mail-from", &sesv2.EmailIdentityMailFromAttributesArgs{
		EmailIdentity:       identity.EmailIdentity,
		MailFromDomain:      pulumi.String(mailFromDomain),
		BehaviorOnMxFailure: pulumi.String("USE_DEFAULT_VALUE"),
	}, pulumi.DependsOn([]pulumi.Resource{mailFromMX, mailFromSPF}))
	if err != nil {
		return nil, err
	}

	// Minimal DMARC policy at the sending domain. `p=none` = monitor-only;
	// we'll tighten to `quarantine` / `reject` once we've watched a few
	// weeks of aggregate reports.
	_, err = route53.NewRecord(ctx, "dmarc", &route53.RecordArgs{
		ZoneId:         zoneID,
		Name:           pulumi.String("_dmarc." + sendingDomain),
		Type:           pulumi.String("TXT"),
		Ttl:            pulumi.Int(300),
		Records:        pulumi.StringArray{pulumi.String("v=DMARC1; p=none; adkim=r; aspf=r")},
		AllowOverwrite: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	// Configuration set: every send goes through this so events can be
	// captured uniformly and reputation can be tracked per logical channel.
	configSet, err := sesv2.NewConfigurationSet(ctx, "events", &sesv2.ConfigurationSetArgs{
		ConfigurationSetName: pulumi.String("the-run-events"),
		ReputationOptions: &sesv2.ConfigurationSetReputationOptionsArgs{
			ReputationMetricsEnabled: pulumi.Bool(true),
		},
		SendingOptions: &sesv2.ConfigurationSetSendingOptionsArgs{
			SendingEnabled: pulumi.Bool(true),
		},
	})
	if err != nil {
		return nil, err
	}

	// Publish bounce / complaint / reject metrics to CloudWatch so we can
	// alarm on poor reputation early. Full bounce payload handling (SNS or
	// Kinesis Firehose with a parser) can come later if the volume needs it.
	_, err = sesv2.NewConfigurationSetEventDestination(ctx, "events-cw", &sesv2.ConfigurationSetEventDestinationArgs{
		ConfigurationSetName: configSet.ConfigurationSetName,
		EventDestinationName: pulumi.String("cloudwatch"),
		EventDestination: &sesv2.ConfigurationSetEventDestinationEventDestinationArgs{
			Enabled: pulumi.Bool(true),
			MatchingEventTypes: pulumi.StringArray{
				pulumi.String("BOUNCE"),
				pulumi.String("COMPLAINT"),
				pulumi.String("REJECT"),
				pulumi.String("DELIVERY_DELAY"),
			},
			CloudWatchDestination: &sesv2.ConfigurationSetEventDestinationEventDestinationCloudWatchDestinationArgs{
				DimensionConfigurations: sesv2.ConfigurationSetEventDestinationEventDestinationCloudWatchDestinationDimensionConfigurationArray{
					&sesv2.ConfigurationSetEventDestinationEventDestinationCloudWatchDestinationDimensionConfigurationArgs{
						// Use a single static dimension so all events land
						// under the same CloudWatch metric stream — we don't
						// yet send through enough distinct senders to need
						// per-tag granularity.
						DimensionName:         pulumi.String("MessageTag"),
						DimensionValueSource:  pulumi.String("MESSAGE_TAG"),
						DefaultDimensionValue: pulumi.String("the-run"),
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return &Resources{
		Identity:            identity,
		ConfigurationSet:    configSet,
		SenderAddress:       senderAddress,
		MailFromDomain:      mailFromDomain,
		ConfigurationSetArn: configSet.Arn,
	}, nil
}

package backend

import (
	"encoding/json"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	awslambda "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/BirgerRydback/the-run/infra/database"
	"github.com/BirgerRydback/the-run/infra/email"
)

// setupRetention provisions the GDPR A1.1.3 retention sweep:
//   - A new Lambda function backed by backend/dist/retention.zip.
//   - Its own IAM role with DynamoDB read+write on the same tables the API
//     Lambda uses (retention purges PII in place, so it needs UpdateItem
//     and DeleteItem). No Secrets Manager or SES — pure data sweep.
//   - Its own CloudWatch Log Group with 30-day retention (A0.5).
//   - An EventBridge rule firing daily at 03:00 UTC (low-traffic window).
func setupRetention(
	ctx *pulumi.Context,
	tables *database.Tables,
	em *email.Resources,
	siteFQDN string,
) error {
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
	role, err := iam.NewRole(ctx, "retention-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(string(assumePolicy)),
	})
	if err != nil {
		return err
	}

	_, err = iam.NewRolePolicyAttachment(ctx, "retention-basic-exec", &iam.RolePolicyAttachmentArgs{
		Role:      role.Name,
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
	})
	if err != nil {
		return err
	}

	// DynamoDB permissions. The retention Lambda reads + writes + deletes
	// across the data tables (sweep semantics: scan, anonymize, delete).
	//
	//   accounts, runners, registrations — soft-delete cleanup + time-based
	//     anonymization
	//   magic tokens                     — drop tokens for purged accounts
	//   events, races                    — read-only, to resolve race dates
	//                                      for the 36-month sliding window
	//   audit                            — read+write to purge rows older
	//                                      than 24 months + write
	//                                      retention.runner.anonymized rows
	dynamoPolicy := pulumi.All(
		tables.Accounts.Arn, tables.Runners.Arn, tables.Registrations.Arn, tables.MagicTokens.Arn,
		tables.Events.Arn, tables.Races.Arn, tables.Audit.Arn, tables.EmailTemplates.Arn,
	).ApplyT(func(args []any) (string, error) {
		accountsArn := args[0].(string)
		runnersArn := args[1].(string)
		regsArn := args[2].(string)
		tokensArn := args[3].(string)
		eventsArn := args[4].(string)
		racesArn := args[5].(string)
		auditArn := args[6].(string)
		emailTemplatesArn := args[7].(string)
		doc, err := json.Marshal(map[string]any{
			"Version": "2012-10-17",
			"Statement": []any{
				map[string]any{
					"Effect": "Allow",
					"Action": []string{
						"dynamodb:GetItem",
						"dynamodb:Query",
						"dynamodb:Scan",
						"dynamodb:PutItem",
						"dynamodb:UpdateItem",
						"dynamodb:DeleteItem",
						"dynamodb:TransactWriteItems",
					},
					"Resource": []string{
						accountsArn,
						accountsArn + "/index/*",
						runnersArn,
						runnersArn + "/index/*",
						regsArn,
						regsArn + "/index/*",
						tokensArn,
						eventsArn,
						racesArn,
						auditArn,
						// Read-only in practice — the renderer GETs the
						// published template row when composing the
						// inactivity-notice email.
						emailTemplatesArn,
					},
				},
			},
		})
		if err != nil {
			return "", err
		}
		return string(doc), nil
	}).(pulumi.StringOutput)

	_, err = iam.NewRolePolicy(ctx, "retention-dynamodb", &iam.RolePolicyArgs{
		Role:   role.ID(),
		Policy: dynamoPolicy,
	})
	if err != nil {
		return err
	}

	// SES send permission. The inactivity sweep emails users whose accounts
	// it's about to soft-delete. Same wildcard-on-identity pattern as the
	// API Lambda — sandbox mode IAM-checks the recipient too, and we can't
	// enumerate future addresses.
	sesPolicy := em.ConfigurationSetArn.ApplyT(func(configSetArn string) (string, error) {
		doc, err := json.Marshal(map[string]any{
			"Version": "2012-10-17",
			"Statement": []any{
				map[string]any{
					"Effect": "Allow",
					"Action": []string{"ses:SendEmail", "ses:SendRawEmail"},
					"Resource": []string{
						"arn:aws:ses:eu-north-1:*:identity/*",
						configSetArn,
					},
				},
			},
		})
		if err != nil {
			return "", err
		}
		return string(doc), nil
	}).(pulumi.StringOutput)

	_, err = iam.NewRolePolicy(ctx, "retention-ses", &iam.RolePolicyArgs{
		Role:   role.ID(),
		Policy: sesPolicy,
	})
	if err != nil {
		return err
	}

	// Dedicated log group with 30-day retention (GDPR A0.5).
	logGroup, err := cloudwatch.NewLogGroup(ctx, "retention-logs", &cloudwatch.LogGroupArgs{
		Name:            pulumi.String("/aws/lambda/retention-sweep"),
		RetentionInDays: pulumi.Int(30),
	})
	if err != nil {
		return err
	}

	fn, err := awslambda.NewFunction(ctx, "retention-fn", &awslambda.FunctionArgs{
		Role:          role.Arn,
		Name:          pulumi.String("retention-sweep"),
		Runtime:       pulumi.String("provided.al2023"),
		Architectures: pulumi.StringArray{pulumi.String("arm64")},
		Handler:       pulumi.String("bootstrap"),
		Code:          pulumi.NewFileArchive("../backend/dist/retention.zip"),
		MemorySize:    pulumi.Int(256),
		// 5 minutes — the sweep iterates every account + runner row. For a
		// <500-runner race that's still <1s in practice, but the upper
		// bound gives headroom if we grow.
		Timeout: pulumi.Int(300),
		Environment: &awslambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap{
				"RUNNERS_TABLE_NAME":          tables.Runners.Name,
				"REGISTRATIONS_TABLE_NAME":    tables.Registrations.Name,
				"EVENTS_TABLE_NAME":           tables.Events.Name,
				"RACES_TABLE_NAME":            tables.Races.Name,
				"ACCOUNTS_TABLE_NAME":         tables.Accounts.Name,
				"AUTH_ATTEMPTS_TABLE_NAME":    tables.AuthAttempts.Name,
				"MAGIC_TOKENS_TABLE_NAME":     tables.MagicTokens.Name,
				"AUDIT_TABLE_NAME":            tables.Audit.Name,
				"RATE_LIMIT_TABLE_NAME":       tables.RateLimit.Name,
				"POLICIES_TABLE_NAME":                 tables.Policies.Name,
				"POLICY_REVISIONS_TABLE_NAME":         tables.PolicyRevisions.Name,
				"EMAIL_TEMPLATES_TABLE_NAME":          tables.EmailTemplates.Name,
				"EMAIL_TEMPLATE_REVISIONS_TABLE_NAME": tables.EmailTemplateRevisions.Name,
				"ADMIN_INVITATIONS_TABLE_NAME":        tables.AdminInvitations.Name,
				// SES + base URL — the inactivity sweep emails a restore
				// link to users about to be auto-deleted.
				"SES_SENDER_ADDRESS":    pulumi.String(em.SenderAddress),
				"SES_CONFIGURATION_SET": em.ConfigurationSet.ConfigurationSetName,
				"SITE_BASE_URL":         pulumi.String("https://" + siteFQDN),
			},
		},
	}, pulumi.DependsOn([]pulumi.Resource{logGroup}))
	if err != nil {
		return err
	}

	// EventBridge daily schedule. 03:00 UTC = ~04:00 / 05:00 in Sweden
	// depending on DST — low traffic, deliberate choice so a slow sweep
	// doesn't compete with race-day registrations.
	rule, err := cloudwatch.NewEventRule(ctx, "retention-schedule", &cloudwatch.EventRuleArgs{
		Name:               pulumi.String("retention-sweep-daily"),
		Description:        pulumi.String("Triggers the GDPR retention Lambda once a day at 03:00 UTC"),
		ScheduleExpression: pulumi.String("cron(0 3 * * ? *)"),
	})
	if err != nil {
		return err
	}

	_, err = cloudwatch.NewEventTarget(ctx, "retention-target", &cloudwatch.EventTargetArgs{
		Rule:     rule.Name,
		Arn:      fn.Arn,
		TargetId: pulumi.String("retention-lambda"),
	})
	if err != nil {
		return err
	}

	_, err = awslambda.NewPermission(ctx, "retention-allow-eventbridge", &awslambda.PermissionArgs{
		Action:    pulumi.String("lambda:InvokeFunction"),
		Function:  fn.Name,
		Principal: pulumi.String("events.amazonaws.com"),
		SourceArn: rule.Arn,
	})
	if err != nil {
		return err
	}

	return nil
}

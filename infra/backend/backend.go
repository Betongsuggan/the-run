// Package backend provisions the Lambda function, API Gateway v2 HTTP API,
// the regional ACM cert + custom domain, and the Route53 alias record pointing
// at the API Gateway domain.
package backend

import (
	"encoding/json"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigatewayv2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	awslambda "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/secretsmanager"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/BirgerRydback/the-run/infra/database"
)

// Resources is the set of backend resources main.go exports outputs for.
type Resources struct {
	API       *apigatewayv2.Api
	Function  *awslambda.Function
	JWTSecret *secretsmanager.Secret
}

// Setup provisions the full backend stack: cert, IAM role, JWT signing secret,
// Lambda, HTTP API, integration + routes + stage + custom domain, and the
// alias record. The Lambda is granted read/write access to the DynamoDB tables
// in `tables`, read-only access to the JWT secret, and receives table names +
// secret ARN via environment variables.
func Setup(
	ctx *pulumi.Context,
	apiFQDN, siteFQDN, cookieDomain string,
	zoneID pulumi.StringInput,
	tables *database.Tables,
) (*Resources, error) {
	cert, certValidation, err := createCert(ctx, apiFQDN, zoneID)
	if err != nil {
		return nil, err
	}

	// Random 64-byte hex string for the JWT signing key. RandomPassword stores
	// the value in Pulumi state encrypted (KMS-backed); rotating means
	// destroy+recreate this resource — invalidates all sessions.
	jwtKey, err := random.NewRandomPassword(ctx, "jwt-signing-key", &random.RandomPasswordArgs{
		Length:  pulumi.Int(64),
		Special: pulumi.Bool(false),
		Upper:   pulumi.Bool(true),
		Lower:   pulumi.Bool(true),
		Numeric: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	jwtSecret, err := secretsmanager.NewSecret(ctx, "jwt-secret", &secretsmanager.SecretArgs{
		Name:        pulumi.String("the-run/jwt-signing-key"),
		Description: pulumi.String("HMAC-SHA256 signing key for the-run admin session JWTs"),
	})
	if err != nil {
		return nil, err
	}

	_, err = secretsmanager.NewSecretVersion(ctx, "jwt-secret-version", &secretsmanager.SecretVersionArgs{
		SecretId:     jwtSecret.ID(),
		SecretString: jwtKey.Result,
	})
	if err != nil {
		return nil, err
	}

	// Cloudflare Turnstile secret key. Stored encrypted; rotation = set a new
	// SecretVersion out-of-band (e.g. `aws secretsmanager put-secret-value`).
	// Initial value is a placeholder so `pulumi up` works before a Cloudflare
	// account exists — the Lambda's LoadConfig logs a warning and skips
	// verification until a real key lands. Update via:
	//   aws secretsmanager put-secret-value \
	//     --secret-id the-run/turnstile-secret-key \
	//     --secret-string '<key-from-cloudflare>'
	turnstileSecret, err := secretsmanager.NewSecret(ctx, "turnstile-secret", &secretsmanager.SecretArgs{
		Name:        pulumi.String("the-run/turnstile-secret-key"),
		Description: pulumi.String("Cloudflare Turnstile secret key for /registrations bot challenge"),
	})
	if err != nil {
		return nil, err
	}

	_, err = secretsmanager.NewSecretVersion(ctx, "turnstile-secret-version", &secretsmanager.SecretVersionArgs{
		SecretId:     turnstileSecret.ID(),
		SecretString: pulumi.String("placeholder-set-via-put-secret-value"),
	})
	if err != nil {
		return nil, err
	}

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
		return nil, err
	}

	lambdaRole, err := iam.NewRole(ctx, "lambda-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(string(assumePolicy)),
	})
	if err != nil {
		return nil, err
	}

	_, err = iam.NewRolePolicyAttachment(ctx, "lambda-basic-exec", &iam.RolePolicyAttachmentArgs{
		Role:      lambdaRole.Name,
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
	})
	if err != nil {
		return nil, err
	}

	dynamoPolicy := pulumi.All(
		tables.Runners.Arn, tables.Registrations.Arn, tables.Events.Arn, tables.Races.Arn,
		tables.Accounts.Arn, tables.AuthAttempts.Arn,
	).ApplyT(func(args []any) (string, error) {
		runnersArn := args[0].(string)
		regsArn := args[1].(string)
		eventsArn := args[2].(string)
		racesArn := args[3].(string)
		accountsArn := args[4].(string)
		attemptsArn := args[5].(string)
		doc, err := json.Marshal(map[string]any{
			"Version": "2012-10-17",
			"Statement": []any{
				map[string]any{
					"Effect": "Allow",
					"Action": []string{
						"dynamodb:GetItem",
						"dynamodb:PutItem",
						"dynamodb:UpdateItem",
						"dynamodb:DeleteItem",
						"dynamodb:Query",
						"dynamodb:Scan",
						"dynamodb:ConditionCheckItem",
						"dynamodb:BatchWriteItem",
						"dynamodb:TransactWriteItems",
					},
					"Resource": []string{
						runnersArn,
						runnersArn + "/index/*",
						regsArn,
						regsArn + "/index/*",
						eventsArn,
						eventsArn + "/index/*",
						racesArn,
						racesArn + "/index/*",
						accountsArn,
						accountsArn + "/index/*",
						attemptsArn,
						attemptsArn + "/index/*",
					},
				},
			},
		})
		if err != nil {
			return "", err
		}
		return string(doc), nil
	}).(pulumi.StringOutput)

	_, err = iam.NewRolePolicy(ctx, "lambda-dynamodb", &iam.RolePolicyArgs{
		Role:   lambdaRole.ID(),
		Policy: dynamoPolicy,
	})
	if err != nil {
		return nil, err
	}

	secretsPolicy := pulumi.All(jwtSecret.Arn, turnstileSecret.Arn).ApplyT(func(args []any) (string, error) {
		jwtArn := args[0].(string)
		turnstileArn := args[1].(string)
		doc, err := json.Marshal(map[string]any{
			"Version": "2012-10-17",
			"Statement": []any{
				map[string]any{
					"Effect":   "Allow",
					"Action":   []string{"secretsmanager:GetSecretValue"},
					"Resource": []string{jwtArn, turnstileArn},
				},
			},
		})
		if err != nil {
			return "", err
		}
		return string(doc), nil
	}).(pulumi.StringOutput)

	_, err = iam.NewRolePolicy(ctx, "lambda-secrets", &iam.RolePolicyArgs{
		Role:   lambdaRole.ID(),
		Policy: secretsPolicy,
	})
	if err != nil {
		return nil, err
	}

	// Pre-create the Lambda's CloudWatch log group so we can pin retention to
	// 30 days (GDPR A0.5). The Lambda runtime writes to
	// `/aws/lambda/<function-name>` regardless; if we don't manage the group,
	// AWS auto-creates it with infinite retention. Note: if a previous deploy
	// already populated this group via auto-create, `pulumi up` will fail with
	// "AlreadyExists" — delete it once with `aws logs delete-log-group
	// --log-group-name /aws/lambda/backend-api` and re-run.
	logGroup, err := cloudwatch.NewLogGroup(ctx, "api-fn-logs", &cloudwatch.LogGroupArgs{
		Name:            pulumi.String("/aws/lambda/backend-api"),
		RetentionInDays: pulumi.Int(30),
	})
	if err != nil {
		return nil, err
	}

	fn, err := awslambda.NewFunction(ctx, "api-fn", &awslambda.FunctionArgs{
		Role:          lambdaRole.Arn,
		Name:          pulumi.String("backend-api"),
		Runtime:       pulumi.String("provided.al2023"),
		Architectures: pulumi.StringArray{pulumi.String("arm64")},
		Handler:       pulumi.String("bootstrap"),
		Code:          pulumi.NewFileArchive("../backend/dist/lambda.zip"),
		MemorySize:    pulumi.Int(256),
		Timeout:       pulumi.Int(10),
		Environment: &awslambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap{
				"RUNNERS_TABLE_NAME":       tables.Runners.Name,
				"REGISTRATIONS_TABLE_NAME": tables.Registrations.Name,
				"EVENTS_TABLE_NAME":        tables.Events.Name,
				"RACES_TABLE_NAME":         tables.Races.Name,
				"ACCOUNTS_TABLE_NAME":      tables.Accounts.Name,
				"AUTH_ATTEMPTS_TABLE_NAME": tables.AuthAttempts.Name,
				"JWT_SECRET_ARN":           jwtSecret.Arn,
				"TURNSTILE_SECRET_ARN":     turnstileSecret.Arn,
				"COOKIE_DOMAIN":            pulumi.String("." + cookieDomain), // common parent so api.<x> and site can share the cookie
				"PRIVACY_POLICY_VERSION":   pulumi.String("2026-08-01"),
			},
		},
	}, pulumi.DependsOn([]pulumi.Resource{logGroup}))
	if err != nil {
		return nil, err
	}

	api, err := apigatewayv2.NewApi(ctx, "api", &apigatewayv2.ApiArgs{
		ProtocolType: pulumi.String("HTTP"),
		CorsConfiguration: &apigatewayv2.ApiCorsConfigurationArgs{
			AllowOrigins: pulumi.StringArray{pulumi.String("https://" + siteFQDN)},
			AllowMethods: pulumi.StringArray{
				pulumi.String("GET"),
				pulumi.String("POST"),
				pulumi.String("PUT"),
				pulumi.String("PATCH"),
				pulumi.String("DELETE"),
				pulumi.String("OPTIONS"),
			},
			AllowHeaders:     pulumi.StringArray{pulumi.String("content-type")},
			AllowCredentials: pulumi.Bool(true),
			MaxAge:           pulumi.Int(300),
		},
	})
	if err != nil {
		return nil, err
	}

	integration, err := apigatewayv2.NewIntegration(ctx, "api-integration", &apigatewayv2.IntegrationArgs{
		ApiId:                api.ID(),
		IntegrationType:      pulumi.String("AWS_PROXY"),
		IntegrationUri:       fn.InvokeArn,
		IntegrationMethod:    pulumi.String("POST"),
		PayloadFormatVersion: pulumi.String("2.0"),
	})
	if err != nil {
		return nil, err
	}

	_, err = apigatewayv2.NewRoute(ctx, "api-route-default", &apigatewayv2.RouteArgs{
		ApiId:    api.ID(),
		RouteKey: pulumi.String("ANY /{proxy+}"),
		Target:   pulumi.Sprintf("integrations/%s", integration.ID()),
	})
	if err != nil {
		return nil, err
	}

	_, err = apigatewayv2.NewRoute(ctx, "api-route-root", &apigatewayv2.RouteArgs{
		ApiId:    api.ID(),
		RouteKey: pulumi.String("ANY /"),
		Target:   pulumi.Sprintf("integrations/%s", integration.ID()),
	})
	if err != nil {
		return nil, err
	}

	// Explicit POST /registrations route so the stage's per-route throttle
	// (below) has something to bind to. More-specific routes win over the
	// `ANY /{proxy+}` catch-all, so this just adds a routing entry; traffic
	// still flows to the same Lambda integration.
	registrationsRoute, err := apigatewayv2.NewRoute(ctx, "api-route-registrations", &apigatewayv2.RouteArgs{
		ApiId:    api.ID(),
		RouteKey: pulumi.String("POST /registrations"),
		Target:   pulumi.Sprintf("integrations/%s", integration.ID()),
	})
	if err != nil {
		return nil, err
	}

	stage, err := apigatewayv2.NewStage(ctx, "api-stage", &apigatewayv2.StageArgs{
		ApiId:      api.ID(),
		Name:       pulumi.String("$default"),
		AutoDeploy: pulumi.Bool(true),
		// Per-route throttling on POST /registrations (GDPR A0.7) — bounds
		// damage from a token-replay or token-farmed flood that gets past
		// Turnstile. Numbers are deliberately conservative: a 500-runner
		// race rarely exceeds ~0.05 req/s in steady state, so 2 sustained
		// + 5 burst leaves >40× headroom for the legitimate sign-up peak
		// while still rejecting a hostile flood. HTTP API throttling is
		// global across all clients (per-IP needs WAF — see A2).
		RouteSettings: apigatewayv2.StageRouteSettingArray{
			&apigatewayv2.StageRouteSettingArgs{
				RouteKey:             pulumi.String("POST /registrations"),
				ThrottlingBurstLimit: pulumi.Int(5),
				ThrottlingRateLimit:  pulumi.Float64(2),
			},
		},
	}, pulumi.DependsOn([]pulumi.Resource{registrationsRoute}))
	if err != nil {
		return nil, err
	}

	_, err = awslambda.NewPermission(ctx, "api-invoke-lambda", &awslambda.PermissionArgs{
		Action:    pulumi.String("lambda:InvokeFunction"),
		Function:  fn.Name,
		Principal: pulumi.String("apigateway.amazonaws.com"),
		SourceArn: pulumi.Sprintf("%s/*/*", api.ExecutionArn),
	})
	if err != nil {
		return nil, err
	}

	apiDomain, err := apigatewayv2.NewDomainName(ctx, "api-domain", &apigatewayv2.DomainNameArgs{
		DomainName: pulumi.String(apiFQDN),
		DomainNameConfiguration: &apigatewayv2.DomainNameDomainNameConfigurationArgs{
			CertificateArn: cert.Arn,
			EndpointType:   pulumi.String("REGIONAL"),
			SecurityPolicy: pulumi.String("TLS_1_2"),
		},
	}, pulumi.DependsOn([]pulumi.Resource{certValidation}))
	if err != nil {
		return nil, err
	}

	_, err = apigatewayv2.NewApiMapping(ctx, "api-mapping", &apigatewayv2.ApiMappingArgs{
		ApiId:      api.ID(),
		DomainName: apiDomain.ID(),
		Stage:      stage.Name,
	})
	if err != nil {
		return nil, err
	}

	if err := createAliasRecord(ctx, zoneID, apiFQDN, apiDomain); err != nil {
		return nil, err
	}

	return &Resources{
		API:       api,
		Function:  fn,
		JWTSecret: jwtSecret,
	}, nil
}

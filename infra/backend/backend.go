// Package backend provisions the Lambda function, API Gateway v2 HTTP API,
// the regional ACM cert + custom domain, and the Route53 alias record pointing
// at the API Gateway domain.
package backend

import (
	"encoding/json"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigatewayv2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	awslambda "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/BirgerRydback/the-run/infra/database"
)

// Resources is the set of backend resources main.go exports outputs for.
type Resources struct {
	API      *apigatewayv2.Api
	Function *awslambda.Function
}

// Setup provisions the full backend stack: cert, IAM role, Lambda, HTTP API,
// integration + routes + stage + custom domain, and the alias record. The
// Lambda is granted read/write access to the DynamoDB tables in `tables` and
// receives their names via environment variables.
func Setup(
	ctx *pulumi.Context,
	apiFQDN, siteFQDN string,
	zoneID pulumi.StringInput,
	tables *database.Tables,
) (*Resources, error) {
	cert, certValidation, err := createCert(ctx, apiFQDN, zoneID)
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
		tables.Runners.Arn, tables.Registrations.Arn,
	).ApplyT(func(args []any) (string, error) {
		runnersArn := args[0].(string)
		regsArn := args[1].(string)
		doc, err := json.Marshal(map[string]any{
			"Version": "2012-10-17",
			"Statement": []any{
				map[string]any{
					"Effect": "Allow",
					"Action": []string{
						"dynamodb:GetItem",
						"dynamodb:PutItem",
						"dynamodb:Query",
						"dynamodb:ConditionCheckItem",
					},
					"Resource": []string{
						runnersArn,
						runnersArn + "/index/*",
						regsArn,
						regsArn + "/index/*",
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
			},
		},
	})
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
				pulumi.String("DELETE"),
				pulumi.String("OPTIONS"),
			},
			AllowHeaders: pulumi.StringArray{pulumi.String("content-type"), pulumi.String("authorization")},
			MaxAge:       pulumi.Int(300),
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

	stage, err := apigatewayv2.NewStage(ctx, "api-stage", &apigatewayv2.StageArgs{
		ApiId:      api.ID(),
		Name:       pulumi.String("$default"),
		AutoDeploy: pulumi.Bool(true),
	})
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
		API:      api,
		Function: fn,
	}, nil
}

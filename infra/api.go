package main

import (
	"encoding/json"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigatewayv2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	awslambda "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type apiResources struct {
	api       *apigatewayv2.Api
	function  *awslambda.Function
	apiDomain *apigatewayv2.DomainName
}

func setupAPI(
	ctx *pulumi.Context,
	apiFQDN, siteFQDN string,
	apiCert *acm.Certificate,
	apiCertValidation *acm.CertificateValidation,
) (*apiResources, error) {
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

	fn, err := awslambda.NewFunction(ctx, "api-fn", &awslambda.FunctionArgs{
		Role:          lambdaRole.Arn,
		Name:          pulumi.String("backend-api"),
		Runtime:       pulumi.String("provided.al2023"),
		Architectures: pulumi.StringArray{pulumi.String("arm64")},
		Handler:       pulumi.String("bootstrap"),
		Code:          pulumi.NewFileArchive("../backend/dist/lambda.zip"),
		MemorySize:    pulumi.Int(256),
		Timeout:       pulumi.Int(10),
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
			CertificateArn: apiCert.Arn,
			EndpointType:   pulumi.String("REGIONAL"),
			SecurityPolicy: pulumi.String("TLS_1_2"),
		},
	}, pulumi.DependsOn([]pulumi.Resource{apiCertValidation}))
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

	return &apiResources{
		api:       api,
		function:  fn,
		apiDomain: apiDomain,
	}, nil
}

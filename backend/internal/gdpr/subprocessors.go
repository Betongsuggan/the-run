package gdpr

// Subprocessor is one row in the ROPA's "Underbiträden" table.
//
// We have an AWS-only stack by policy (see feedback_aws_only in the memory
// folder + the PROJECT_PLAN's note). If you add a non-AWS dependency,
// append it here and the ROPA picks it up automatically.
type Subprocessor struct {
	Name     string
	Role     string
	Region   string
	Services []string
	DPA      string
}

var Subprocessors = []Subprocessor{
	{
		Name:     "Amazon Web Services EMEA SARL",
		Role:     "Drift, lagring och e-postutskick",
		Region:   "eu-north-1 (Stockholm) + CloudFront-edge inom EES",
		Services: []string{"DynamoDB", "Lambda", "API Gateway", "CloudFront", "SES", "Secrets Manager", "CloudWatch Logs"},
		DPA:      "Accepterad via AWS Artifact",
	},
}

package observability

import (
	awsconfig "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
)

// ConfigureXRay sets up X-Ray with the default daemon address.
func ConfigureXRay() error {
	return xray.Configure(xray.Config{
		LogLevel: "info",
	})
}

// InstrumentAWSConfig instruments an AWS SDK v2 config for X-Ray tracing.
func InstrumentAWSConfig(cfg *awsconfig.Config) {
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)
}

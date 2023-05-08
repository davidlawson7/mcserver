package bot

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

// Loads a default context and invokes a lambda function with the specified method (start, stop)
// Returns the results or a error.
func InvokeLambda(ctx *context.Context, method string) (*lambda.InvokeOutput, error) {
	// AWS CONFIG
	cfg, err := config.LoadDefaultConfig(*ctx,
		config.WithRegion("ap-southeast-2"),
	)

	if err != nil {
		fmt.Println(err)
		return &lambda.InvokeOutput{}, err
	}

	// AWS LAMBDA
	lmb := lambda.NewFromConfig(cfg)

	return lmb.Invoke(*ctx, &lambda.InvokeInput{
		FunctionName: aws.String("mc_operations"),
		Payload:      []byte(*aws.String(fmt.Sprintf(`{"requestType": "%s"}`, method))), // You could pass Request here
	})
}

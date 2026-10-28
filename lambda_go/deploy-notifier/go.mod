module github.com/satoshif1977/aws-cicd-ecs-pipeline/lambda_go/deploy-notifier

go 1.26

require (
	github.com/aws/aws-lambda-go v1.55.1
	github.com/aws/aws-sdk-go-v2 v1.47.4
	github.com/aws/aws-sdk-go-v2/service/sns v1.47.6
	github.com/aws/smithy-go v1.28.6
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.7 // indirect
)

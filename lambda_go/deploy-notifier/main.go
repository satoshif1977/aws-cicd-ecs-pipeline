// Package main は ECS デプロイ完了通知 Lambda（Go 実装）。
//
// GitHub Actions から ECS デプロイ完了後に呼び出され、
// デプロイ結果（サービス名・タスク定義・ステータス）を
// SNS トピックへ通知する。
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// ── インターフェース ───────────────────────────────────────────

// SNSPublisher は SNS Publish 操作を抽象化するインターフェース。
type SNSPublisher interface {
	Publish(ctx context.Context, input *sns.PublishInput, opts ...func(*sns.Options)) (*sns.PublishOutput, error)
}

// ── イベント / レスポンス ─────────────────────────────────────

// DeployEvent は GitHub Actions から渡されるデプロイ情報。
type DeployEvent struct {
	Service        string `json:"service"`
	Cluster        string `json:"cluster"`
	TaskDefinition string `json:"task_definition"`
	ImageTag       string `json:"image_tag"`
	Status         string `json:"status"` // "success" | "failure"
	DeployedAt     string `json:"deployed_at,omitempty"`
}

// NotifyResult は Lambda のレスポンス。
type NotifyResult struct {
	MessageID string `json:"message_id"`
	Message   string `json:"message"`
}

// ── ハンドラー ────────────────────────────────────────────────

// Handler は DeployEvent を受け取り SNS へ通知する。
func Handler(publisher SNSPublisher, topicArn string) func(ctx context.Context, event DeployEvent) (NotifyResult, error) {
	return func(ctx context.Context, event DeployEvent) (NotifyResult, error) {
		if event.DeployedAt == "" {
			event.DeployedAt = time.Now().UTC().Format(time.RFC3339)
		}

		emoji := "✅"
		if event.Status != "success" {
			emoji = "❌"
		}

		message := fmt.Sprintf(
			"%s ECS Deploy Notification\n\nCluster:  %s\nService:  %s\nTask Def: %s\nImage:    %s\nStatus:   %s\nTime:     %s",
			emoji,
			event.Cluster,
			event.Service,
			event.TaskDefinition,
			event.ImageTag,
			event.Status,
			event.DeployedAt,
		)

		subject := fmt.Sprintf("[ECS] Deploy %s - %s", event.Status, event.Service)

		out, err := publisher.Publish(ctx, &sns.PublishInput{
			TopicArn: aws.String(topicArn),
			Subject:  aws.String(subject),
			Message:  aws.String(message),
		})
		if err != nil {
			return NotifyResult{}, fmt.Errorf("sns publish failed: %w", err)
		}

		return NotifyResult{
			MessageID: aws.ToString(out.MessageId),
			Message:   message,
		}, nil
	}
}

// ── エントリーポイント ────────────────────────────────────────

func main() {
	topicArn := os.Getenv("SNS_TOPIC_ARN")
	if topicArn == "" {
		panic("SNS_TOPIC_ARN is required")
	}

	cfg := aws.Config{Region: os.Getenv("AWS_REGION")}
	if cfg.Region == "" {
		cfg.Region = "ap-northeast-1"
	}

	client := sns.NewFromConfig(cfg)
	lambda.Start(Handler(client, topicArn))
}

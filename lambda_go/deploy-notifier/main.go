// Package main は ECS デプロイ完了通知 Lambda（Go 実装）。
//
// GitHub Actions から ECS デプロイ完了後に呼び出され、
// デプロイ結果（サービス名・タスク定義・ステータス）を
// SNS トピックへ通知する。
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// ── リトライ / ログ ───────────────────────────────────────────

// RetryOperation はリトライのログ・計測に使う操作名。
const RetryOperation = "Publish"

// retrier は SNS Publish に使うリトライ実行器（retry.go を参照）。
// 値型なので、フックを差すときは呼び出しごとにコピーする。
// テストからは Sleep / Config を差し替えて実待機ゼロにできる。
var retrier = NewRetrier()

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
//
// logger が nil の場合は環境変数から既定のロガーを組み立てる
// （既存の呼び出し側を壊さないため、引数は増やさず内部で補う）。
func Handler(publisher SNSPublisher, topicArn string) func(ctx context.Context, event DeployEvent) (NotifyResult, error) {
	return HandlerWithLogger(publisher, topicArn, nil)
}

// HandlerWithLogger は Handler にロガーを明示できる版。
func HandlerWithLogger(publisher SNSPublisher, topicArn string, logger *slog.Logger) func(ctx context.Context, event DeployEvent) (NotifyResult, error) {
	if logger == nil {
		logger = NewLoggerFromEnv(LoggerOptions{})
	}
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

		// リトライ層へログフックを差し込む（Retrier は値型なのでコピーしてから設定）
		r := retrier
		r.OnRetry = RetryLogHook(logger, RetryOperation)

		out, err := RetryValue(ctx, r, RetryOperation, func(c context.Context) (*sns.PublishOutput, error) {
			return publisher.Publish(c, &sns.PublishInput{
				TopicArn: aws.String(topicArn),
				Subject:  aws.String(subject),
				Message:  aws.String(message),
			})
		})
		if err != nil {
			logger.Error("SNS への通知に失敗しました",
				"operation", RetryOperation,
				"service", event.Service,
				"status", event.Status,
				"error", err,
			)
			return NotifyResult{}, fmt.Errorf("sns publish failed: %w", err)
		}

		logger.Info("デプロイ通知を送信しました",
			"operation", RetryOperation,
			"service", event.Service,
			"cluster", event.Cluster,
			"status", event.Status,
			"messageId", aws.ToString(out.MessageId),
		)

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

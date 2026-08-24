package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// ── captureMock: メッセージキャプチャ用ヘルパー ─────────────────────

func captureMock() (*mockSNS, *sns.PublishInput) {
	var captured *sns.PublishInput
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = input
			return &sns.PublishOutput{MessageId: aws.String("cap-id")}, nil
		},
	}
	return mock, captured
}

// captureHandler は Handler を呼び出して、キャプチャした PublishInput を返す。
func captureHandler(event DeployEvent) (*sns.PublishInput, NotifyResult, error) {
	var captured *sns.PublishInput
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = input
			return &sns.PublishOutput{MessageId: aws.String("cap-id")}, nil
		},
	}
	h := Handler(mock, "arn:aws:sns:ap-northeast-1:123:test")
	result, err := h(context.Background(), event)
	return captured, result, err
}

// ── メッセージ行構造テスト ──────────────────────────────────────────

func TestHandler_MessageLineCount(t *testing.T) {
	input, _, err := captureHandler(DeployEvent{
		Service:        "svc",
		Cluster:        "cluster",
		TaskDefinition: "task:1",
		ImageTag:       "tag",
		Status:         "success",
		DeployedAt:     "2026-08-24T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(aws.ToString(input.Message), "\n")
	// フォーマット: emoji行 + 空行 + Cluster + Service + Task Def + Image + Status + Time = 8行
	if len(lines) != 8 {
		t.Errorf("message should have 8 lines, got %d:\n%s", len(lines), aws.ToString(input.Message))
	}
}

func TestHandler_MessageFirstLine(t *testing.T) {
	input, _, err := captureHandler(DeployEvent{
		Service: "svc", Status: "success",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(aws.ToString(input.Message), "\n")
	if !strings.Contains(lines[0], "ECS Deploy Notification") {
		t.Errorf("first line should contain 'ECS Deploy Notification', got %q", lines[0])
	}
}

func TestHandler_MessageSecondLineEmpty(t *testing.T) {
	input, _, err := captureHandler(DeployEvent{
		Service: "svc", Status: "success",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(aws.ToString(input.Message), "\n")
	if len(lines) < 2 {
		t.Fatal("message should have at least 2 lines")
	}
	if lines[1] != "" {
		t.Errorf("second line should be empty, got %q", lines[1])
	}
}

// ── Result.Message = SNS メッセージ 一致検証 ────────────────────────

func TestHandler_ResultMessageMatchesSNS(t *testing.T) {
	input, result, err := captureHandler(DeployEvent{
		Service:        "api",
		Cluster:        "prod",
		TaskDefinition: "api:10",
		ImageTag:       "v2.0",
		Status:         "success",
		DeployedAt:     "2026-08-24T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != aws.ToString(input.Message) {
		t.Errorf("result.Message should match SNS message:\nresult: %q\nSNS:    %q", result.Message, aws.ToString(input.Message))
	}
}

// ── errors.Is によるエラーラップ検証 ────────────────────────────────

func TestHandler_ErrorWrapsOriginal(t *testing.T) {
	originalErr := errors.New("network timeout")
	mock := &mockSNS{
		publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return nil, originalErr
		},
	}
	h := Handler(mock, "arn:aws:sns:ap-northeast-1:123:test")
	_, err := h(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if !errors.Is(err, originalErr) {
		t.Errorf("error should wrap original: got %v", err)
	}
}

func TestHandler_ErrorOnFailureStatus(t *testing.T) {
	mock := &mockSNS{
		publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return nil, errors.New("fail")
		},
	}
	h := Handler(mock, "arn:aws:sns:ap-northeast-1:123:test")
	_, err := h(context.Background(), DeployEvent{Service: "svc", Status: "failure"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "sns publish failed") {
		t.Errorf("error should contain 'sns publish failed': %v", err)
	}
}

// ── クロージャの topicArn 保持検証 ──────────────────────────────────

func TestHandler_ClosureCapturesTopicArn(t *testing.T) {
	arns := []string{
		"arn:aws:sns:ap-northeast-1:111:topic-a",
		"arn:aws:sns:us-east-1:222:topic-b",
	}
	for _, arn := range arns {
		t.Run(arn, func(t *testing.T) {
			var captured string
			mock := &mockSNS{
				publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
					captured = aws.ToString(input.TopicArn)
					return &sns.PublishOutput{MessageId: aws.String("x")}, nil
				},
			}
			h := Handler(mock, arn)
			h(context.Background(), DeployEvent{Service: "svc", Status: "success"})
			if captured != arn {
				t.Errorf("TopicArn = %q, want %q", captured, arn)
			}
		})
	}
}

// ── Subject フォーマット テーブル駆動（拡張） ───────────────────────

func TestHandler_SubjectFormat_Table(t *testing.T) {
	tests := []struct {
		service string
		status  string
		want    string
	}{
		{"web-api", "success", "[ECS] Deploy success - web-api"},
		{"worker", "failure", "[ECS] Deploy failure - worker"},
		{"batch", "unknown", "[ECS] Deploy unknown - batch"},
		{"", "success", "[ECS] Deploy success - "},
		{"svc", "", "[ECS] Deploy  - svc"},
		{"checkout-api", "rollback", "[ECS] Deploy rollback - checkout-api"},
	}
	for _, tt := range tests {
		t.Run(tt.service+"/"+tt.status, func(t *testing.T) {
			var captured string
			mock := &mockSNS{
				publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
					captured = aws.ToString(input.Subject)
					return &sns.PublishOutput{MessageId: aws.String("x")}, nil
				},
			}
			h := Handler(mock, "arn:aws:sns:ap-northeast-1:123:test")
			h(context.Background(), DeployEvent{Service: tt.service, Status: tt.status})
			if captured != tt.want {
				t.Errorf("subject = %q, want %q", captured, tt.want)
			}
		})
	}
}

// ── 絵文字ロジック テーブル駆動（拡張） ─────────────────────────────

func TestHandler_EmojiLogic_Table(t *testing.T) {
	tests := []struct {
		status string
		emoji  string
	}{
		{"success", "✅"},
		{"failure", "❌"},
		{"unknown", "❌"},
		{"", "❌"},
		{"rollback", "❌"},
		{"pending", "❌"},
		{"SUCCESS", "❌"}, // 大文字は success ではない
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			input, _, err := captureHandler(DeployEvent{Service: "svc", Status: tt.status})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			msg := aws.ToString(input.Message)
			if !strings.Contains(msg, tt.emoji) {
				t.Errorf("status=%q: expected %s, got message: %s", tt.status, tt.emoji, msg)
			}
		})
	}
}

// ── DeployedAt のデフォルト値検証 ───────────────────────────────────

func TestHandler_DeployedAtDefault_ContainsT(t *testing.T) {
	input, _, err := captureHandler(DeployEvent{Service: "svc", Status: "success"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg := aws.ToString(input.Message)
	// DeployedAt 省略時は time.Now().UTC().Format(RFC3339) → "T" を含む
	lines := strings.Split(msg, "\n")
	timeLine := lines[len(lines)-1] // 最後の行が Time
	if !strings.Contains(timeLine, "T") {
		t.Errorf("default DeployedAt should contain 'T' (RFC3339), got: %q", timeLine)
	}
}

func TestHandler_DeployedAtExplicit_Preserved(t *testing.T) {
	input, _, err := captureHandler(DeployEvent{
		Service:    "svc",
		Status:     "success",
		DeployedAt: "2000-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg := aws.ToString(input.Message)
	if !strings.Contains(msg, "2000-01-01T00:00:00Z") {
		t.Errorf("explicit DeployedAt should be preserved, got: %s", msg)
	}
}

// ── メッセージラベル一括検証 ────────────────────────────────────────

func TestHandler_AllLabelsPresent(t *testing.T) {
	input, _, err := captureHandler(DeployEvent{
		Service:        "svc",
		Cluster:        "cluster",
		TaskDefinition: "task:1",
		ImageTag:       "tag",
		Status:         "success",
		DeployedAt:     "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg := aws.ToString(input.Message)
	labels := []string{"Cluster:", "Service:", "Task Def:", "Image:", "Status:", "Time:"}
	for _, label := range labels {
		if !strings.Contains(msg, label) {
			t.Errorf("message should contain label %q", label)
		}
	}
}

// ── メッセージ全フィールド + Subject 一括テスト ─────────────────────

func TestHandler_FullIntegration(t *testing.T) {
	event := DeployEvent{
		Service:        "payment-api",
		Cluster:        "prod-cluster",
		TaskDefinition: "payment:25",
		ImageTag:       "git-abc123",
		Status:         "success",
		DeployedAt:     "2026-08-24T15:30:00Z",
	}
	input, result, err := captureHandler(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Subject
	subject := aws.ToString(input.Subject)
	if subject != "[ECS] Deploy success - payment-api" {
		t.Errorf("subject = %q", subject)
	}

	// Message に全フィールドが含まれる
	msg := aws.ToString(input.Message)
	required := []string{
		"✅", "ECS Deploy Notification",
		"prod-cluster", "payment-api", "payment:25",
		"git-abc123", "success", "2026-08-24T15:30:00Z",
	}
	for _, r := range required {
		if !strings.Contains(msg, r) {
			t.Errorf("message missing %q", r)
		}
	}

	// Result
	if result.MessageID != "cap-id" {
		t.Errorf("MessageID = %q, want cap-id", result.MessageID)
	}
	if result.Message != msg {
		t.Error("result.Message should match SNS message")
	}
}

func TestHandler_FullIntegration_Failure(t *testing.T) {
	event := DeployEvent{
		Service:        "worker",
		Cluster:        "staging",
		TaskDefinition: "worker:3",
		ImageTag:       "rc-1",
		Status:         "failure",
		DeployedAt:     "2026-08-24T16:00:00Z",
	}
	input, _, err := captureHandler(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	subject := aws.ToString(input.Subject)
	if subject != "[ECS] Deploy failure - worker" {
		t.Errorf("subject = %q", subject)
	}

	msg := aws.ToString(input.Message)
	if !strings.Contains(msg, "❌") {
		t.Error("failure message should contain ❌")
	}
	if strings.Contains(msg, "✅") {
		t.Error("failure message should NOT contain ✅")
	}
}

// ── ベンチマーク ────────────────────────────────────────────────────

func BenchmarkHandler_Success(b *testing.B) {
	mock := &mockSNS{
		publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return &sns.PublishOutput{MessageId: aws.String("bench")}, nil
		},
	}
	h := Handler(mock, "arn:aws:sns:ap-northeast-1:123:bench")
	event := DeployEvent{
		Service:        "bench-svc",
		Cluster:        "bench-cluster",
		TaskDefinition: "bench-task:1",
		ImageTag:       "bench-tag",
		Status:         "success",
		DeployedAt:     "2026-08-24T00:00:00Z",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(context.Background(), event)
	}
}

func BenchmarkHandler_Failure(b *testing.B) {
	mock := &mockSNS{
		publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return &sns.PublishOutput{MessageId: aws.String("bench")}, nil
		},
	}
	h := Handler(mock, "arn:aws:sns:ap-northeast-1:123:bench")
	event := DeployEvent{
		Service: "bench-svc",
		Status:  "failure",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(context.Background(), event)
	}
}

package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// ── モック ────────────────────────────────────────────────────

type mockSNS struct {
	publishFunc func(ctx context.Context, input *sns.PublishInput, opts ...func(*sns.Options)) (*sns.PublishOutput, error)
}

func (m *mockSNS) Publish(ctx context.Context, input *sns.PublishInput, opts ...func(*sns.Options)) (*sns.PublishOutput, error) {
	return m.publishFunc(ctx, input, opts...)
}

// ── テスト ────────────────────────────────────────────────────

func TestHandler_Success(t *testing.T) {
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			if aws.ToString(input.TopicArn) != "arn:aws:sns:ap-northeast-1:123456789012:deploy-notify" {
				t.Errorf("unexpected TopicArn: %s", aws.ToString(input.TopicArn))
			}
			if aws.ToString(input.Subject) != "[ECS] Deploy success - my-service" {
				t.Errorf("unexpected Subject: %s", aws.ToString(input.Subject))
			}
			return &sns.PublishOutput{MessageId: aws.String("msg-001")}, nil
		},
	}

	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:deploy-notify")
	result, err := handler(context.Background(), DeployEvent{
		Service:        "my-service",
		Cluster:        "my-cluster",
		TaskDefinition: "my-task:5",
		ImageTag:       "abc1234",
		Status:         "success",
		DeployedAt:     "2026-06-17T00:00:00Z",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MessageID != "msg-001" {
		t.Errorf("expected msg-001, got %s", result.MessageID)
	}
}

func TestHandler_Failure_Status(t *testing.T) {
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			if aws.ToString(input.Subject) != "[ECS] Deploy failure - my-service" {
				t.Errorf("unexpected Subject: %s", aws.ToString(input.Subject))
			}
			return &sns.PublishOutput{MessageId: aws.String("msg-002")}, nil
		},
	}

	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:deploy-notify")
	result, err := handler(context.Background(), DeployEvent{
		Service: "my-service",
		Cluster: "my-cluster",
		Status:  "failure",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MessageID != "msg-002" {
		t.Errorf("expected msg-002, got %s", result.MessageID)
	}
}

func TestHandler_SNS_Error(t *testing.T) {
	mock := &mockSNS{
		publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return nil, errors.New("sns unavailable")
		},
	}

	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:deploy-notify")
	_, err := handler(context.Background(), DeployEvent{
		Service: "my-service",
		Status:  "success",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHandler_DefaultTimestamp(t *testing.T) {
	mock := &mockSNS{
		publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return &sns.PublishOutput{MessageId: aws.String("msg-003")}, nil
		},
	}

	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:deploy-notify")
	// DeployedAt を省略 → デフォルトで現在時刻が入る
	result, err := handler(context.Background(), DeployEvent{
		Service: "my-service",
		Status:  "success",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MessageID != "msg-003" {
		t.Errorf("expected msg-003, got %s", result.MessageID)
	}
}

// ── メッセージ内容検証 ────────────────────────────────────────

func TestHandler_MessageContainsServiceName(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, err := handler(context.Background(), DeployEvent{Service: "web-api", Cluster: "main", Status: "success"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(captured, "web-api") {
		t.Errorf("message should contain service name, got: %s", captured)
	}
}

func TestHandler_MessageContainsCluster(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, err := handler(context.Background(), DeployEvent{Service: "svc", Cluster: "prod-cluster", Status: "success"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(captured, "prod-cluster") {
		t.Errorf("message should contain cluster name, got: %s", captured)
	}
}

func TestHandler_MessageContainsTaskDefinition(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, err := handler(context.Background(), DeployEvent{Service: "svc", TaskDefinition: "api-task:42", Status: "success"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(captured, "api-task:42") {
		t.Errorf("message should contain task definition, got: %s", captured)
	}
}

func TestHandler_MessageContainsImageTag(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, err := handler(context.Background(), DeployEvent{Service: "svc", ImageTag: "sha256abc", Status: "success"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(captured, "sha256abc") {
		t.Errorf("message should contain image tag, got: %s", captured)
	}
}

func TestHandler_MessageContainsECSDeployNotification(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, err := handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(captured, "ECS Deploy Notification") {
		t.Errorf("message should contain 'ECS Deploy Notification', got: %s", captured)
	}
}

// ── 絵文字ロジック ────────────────────────────────────────────

func TestHandler_SuccessEmoji(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if !strings.Contains(captured, "✅") {
		t.Errorf("success message should contain ✅, got: %s", captured)
	}
}

func TestHandler_FailureEmoji(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "failure"})
	if !strings.Contains(captured, "❌") {
		t.Errorf("failure message should contain ❌, got: %s", captured)
	}
}

func TestHandler_TableDrivenStatusEmoji(t *testing.T) {
	cases := []struct {
		status       string
		expectedEmoji string
	}{
		{"success", "✅"},
		{"failure", "❌"},
		{"unknown", "❌"},
		{"", "❌"},
	}

	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			var captured string
			mock := &mockSNS{
				publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
					captured = aws.ToString(input.Message)
					return &sns.PublishOutput{MessageId: aws.String("x")}, nil
				},
			}
			handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
			_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: tc.status})
			if !strings.Contains(captured, tc.expectedEmoji) {
				t.Errorf("status=%q: expected %s in message, got: %s", tc.status, tc.expectedEmoji, captured)
			}
		})
	}
}

// ── Subject / TopicArn 検証 ───────────────────────────────────

func TestHandler_SubjectContainsStatus(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Subject)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if !strings.Contains(captured, "success") {
		t.Errorf("subject should contain status, got: %s", captured)
	}
}

func TestHandler_SubjectContainsServiceName(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Subject)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "checkout-api", Status: "success"})
	if !strings.Contains(captured, "checkout-api") {
		t.Errorf("subject should contain service name, got: %s", captured)
	}
}

func TestHandler_TopicArnPassedToSNS(t *testing.T) {
	const wantArn = "arn:aws:sns:ap-northeast-1:999999999999:my-topic"
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.TopicArn)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, wantArn)
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if captured != wantArn {
		t.Errorf("expected TopicArn=%s, got %s", wantArn, captured)
	}
}

// ── エラー / 結果検証 ─────────────────────────────────────────

func TestHandler_ErrorWrapsMessage(t *testing.T) {
	mock := &mockSNS{
		publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return nil, errors.New("connection timeout")
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, err := handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "sns publish failed") {
		t.Errorf("error should contain 'sns publish failed', got: %v", err)
	}
}

func TestHandler_ResultMessageNotEmpty(t *testing.T) {
	mock := &mockSNS{
		publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	result, err := handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message == "" {
		t.Error("result.Message should not be empty")
	}
}

func TestHandler_DeployedAtPreserved(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{
		Service:    "svc",
		Status:     "success",
		DeployedAt: "2026-01-15T12:00:00Z",
	})
	if !strings.Contains(captured, "2026-01-15T12:00:00Z") {
		t.Errorf("message should contain the specified DeployedAt, got: %s", captured)
	}
}

// ── メッセージ フォーマット詳細 ───────────────────────────────

func TestHandler_MessageContainsStatus(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if !strings.Contains(captured, "success") {
		t.Errorf("message should contain status string, got: %s", captured)
	}
}

func TestHandler_SubjectHasECSPrefix(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Subject)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if !strings.HasPrefix(captured, "[ECS]") {
		t.Errorf("subject should start with [ECS], got: %s", captured)
	}
}

func TestHandler_AllFieldsInMessage(t *testing.T) {
	// 全フィールドが1つのメッセージに含まれること
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{
		Service:        "all-fields-svc",
		Cluster:        "all-fields-cluster",
		TaskDefinition: "all-fields-task:7",
		ImageTag:       "tag-abc",
		Status:         "success",
		DeployedAt:     "2026-07-09T00:00:00Z",
	})
	for _, want := range []string{"all-fields-svc", "all-fields-cluster", "all-fields-task:7", "tag-abc", "2026-07-09T00:00:00Z"} {
		if !strings.Contains(captured, want) {
			t.Errorf("message should contain %q, got: %s", want, captured)
		}
	}
}

func TestHandler_SNSPublishCalledOnce(t *testing.T) {
	// 1 イベントにつき SNS Publish は 1 回だけ呼ばれること
	callCount := 0
	mock := &mockSNS{
		publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			callCount++
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if callCount != 1 {
		t.Errorf("expected SNS Publish to be called once, got %d", callCount)
	}
}

func TestHandler_TableDrivenEvents(t *testing.T) {
	// 多様なイベントパターンでもエラーにならないこと
	cases := []struct {
		name  string
		event DeployEvent
	}{
		{"success-all-fields", DeployEvent{Service: "api", Cluster: "main", TaskDefinition: "api:1", ImageTag: "sha1", Status: "success"}},
		{"failure-minimal", DeployEvent{Service: "worker", Status: "failure"}},
		{"empty-cluster", DeployEvent{Service: "svc", Status: "success", Cluster: ""}},
		{"empty-image-tag", DeployEvent{Service: "svc", Status: "success", ImageTag: ""}},
		{"empty-task-def", DeployEvent{Service: "svc", Status: "success", TaskDefinition: ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockSNS{
				publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
					return &sns.PublishOutput{MessageId: aws.String("x")}, nil
				},
			}
			handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
			_, err := handler(context.Background(), tc.event)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestHandler_MessageContainsClusterLabel(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Cluster: "prod", Status: "success"})
	if !strings.Contains(captured, "Cluster:") {
		t.Errorf("message should contain 'Cluster:' label, got: %s", captured)
	}
}

func TestHandler_MessageContainsNewline(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if !strings.Contains(captured, "\n") {
		t.Errorf("message should contain newlines for readability, got: %s", captured)
	}
}

func TestHandler_TableDrivenSubjectFormat(t *testing.T) {
	// subject のフォーマット "[ECS] Deploy {status} - {service}" を全パターン検証
	cases := []struct {
		service string
		status  string
		want    string
	}{
		{"web-api", "success", "[ECS] Deploy success - web-api"},
		{"payment", "failure", "[ECS] Deploy failure - payment"},
		{"worker", "unknown", "[ECS] Deploy unknown - worker"},
	}
	for _, tc := range cases {
		t.Run(tc.service+"-"+tc.status, func(t *testing.T) {
			var captured string
			mock := &mockSNS{
				publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
					captured = aws.ToString(input.Subject)
					return &sns.PublishOutput{MessageId: aws.String("x")}, nil
				},
			}
			handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
			_, _ = handler(context.Background(), DeployEvent{Service: tc.service, Status: tc.status})
			if captured != tc.want {
				t.Errorf("expected subject=%q, got %q", tc.want, captured)
			}
		})
	}
}

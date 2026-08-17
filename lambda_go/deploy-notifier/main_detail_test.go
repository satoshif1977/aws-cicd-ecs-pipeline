package main

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// ── メッセージラベル確認テスト ────────────────────────────────

func TestHandler_MessageContainsServiceLabel(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if !strings.Contains(captured, "Service:") {
		t.Errorf("message should contain 'Service:' label, got: %s", captured)
	}
}

func TestHandler_MessageContainsTaskDefLabel(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if !strings.Contains(captured, "Task Def:") {
		t.Errorf("message should contain 'Task Def:' label, got: %s", captured)
	}
}

func TestHandler_MessageContainsImageLabel(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if !strings.Contains(captured, "Image:") {
		t.Errorf("message should contain 'Image:' label, got: %s", captured)
	}
}

func TestHandler_MessageContainsTimeLabel(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if !strings.Contains(captured, "Time:") {
		t.Errorf("message should contain 'Time:' label, got: %s", captured)
	}
}

func TestHandler_MessageContainsStatusLabel(t *testing.T) {
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if !strings.Contains(captured, "Status:") {
		t.Errorf("message should contain 'Status:' label, got: %s", captured)
	}
}

// ── 結果検証テスト ────────────────────────────────────────────

func TestHandler_MessageIDMatchesSNSOutput(t *testing.T) {
	// result.MessageID が SNS の MessageId と一致すること
	mock := &mockSNS{
		publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return &sns.PublishOutput{MessageId: aws.String("unique-msg-id-9999")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	result, err := handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MessageID != "unique-msg-id-9999" {
		t.Errorf("MessageID: want unique-msg-id-9999, got %s", result.MessageID)
	}
}

func TestHandler_DefaultTimestampRFC3339(t *testing.T) {
	// DeployedAt 省略時にデフォルト値が RFC3339 形式でメッセージに含まれること
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
	// RFC3339 形式（"T" と "Z" を含む）が出力されること
	if !strings.Contains(captured, "T") || !strings.Contains(captured, "Z") {
		t.Errorf("default timestamp should be RFC3339 (contains T and Z), got: %s", captured)
	}
}

// ── エッジケーステスト ────────────────────────────────────────

func TestHandler_EmptyServiceName(t *testing.T) {
	// Service が空でもエラーにならない（SNS 通知は実行される）
	mock := &mockSNS{
		publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, err := handler(context.Background(), DeployEvent{Service: "", Status: "success"})
	if err != nil {
		t.Errorf("empty service name should not cause error, got: %v", err)
	}
}

func TestHandler_FailureStatusInMessage(t *testing.T) {
	// failure 時にメッセージ本文に "failure" が含まれること
	var captured string
	mock := &mockSNS{
		publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			captured = aws.ToString(input.Message)
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	_, _ = handler(context.Background(), DeployEvent{Service: "svc", Status: "failure"})
	if !strings.Contains(captured, "failure") {
		t.Errorf("failure message should contain 'failure', got: %s", captured)
	}
}

func TestHandler_MultipleCallsSameHandler(t *testing.T) {
	// 同じ handler クロージャを複数回呼んでも問題ないこと
	callCount := 0
	mock := &mockSNS{
		publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
			callCount++
			return &sns.PublishOutput{MessageId: aws.String("x")}, nil
		},
	}
	handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
	for i := 0; i < 3; i++ {
		_, err := handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i+1, err)
		}
	}
	if callCount != 3 {
		t.Errorf("expected 3 SNS calls, got %d", callCount)
	}
}

// ── Fuzz テスト ───────────────────────────────────────────────

func FuzzHandlerEvent(f *testing.F) {
	seeds := []struct{ service, cluster, taskDef, imageTag, status, deployedAt string }{
		{"my-service", "my-cluster", "my-task:5", "abc1234", "success", "2026-06-17T00:00:00Z"},
		{"worker", "prod", "worker:10", "sha256abc", "failure", ""},
		{"api", "", "", "", "success", ""},
		{"", "cluster", "task:1", "tag", "unknown", ""},
		{"svc", "cluster", "task:99", "latest", "", "2026-01-01T00:00:00Z"},
		{"a", "b", "c", "d", "success", "2026-12-31T23:59:59Z"},
		{"long-service-name-test", "long-cluster-name", "task-def:100", "sha-long", "failure", ""},
		{"svc", "cluster", "task:1", "tag", "pending", ""},
		{"checkout-api", "prod-cluster", "checkout:42", "v1.2.3", "success", ""},
		{"payment", "staging", "payment:7", "rc-1", "failure", "2026-08-18T09:00:00Z"},
	}
	for _, s := range seeds {
		f.Add(s.service, s.cluster, s.taskDef, s.imageTag, s.status, s.deployedAt)
	}
	f.Fuzz(func(t *testing.T, service, cluster, taskDef, imageTag, status, deployedAt string) {
		if !utf8.ValidString(service) || !utf8.ValidString(cluster) ||
			!utf8.ValidString(taskDef) || !utf8.ValidString(imageTag) ||
			!utf8.ValidString(status) || !utf8.ValidString(deployedAt) {
			t.Skip()
		}
		mock := &mockSNS{
			publishFunc: func(_ context.Context, _ *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
				return &sns.PublishOutput{MessageId: aws.String("fuzz-msg-id")}, nil
			},
		}
		handler := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:test")
		result, err := handler(context.Background(), DeployEvent{
			Service:        service,
			Cluster:        cluster,
			TaskDefinition: taskDef,
			ImageTag:       imageTag,
			Status:         status,
			DeployedAt:     deployedAt,
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result.MessageID != "fuzz-msg-id" {
			t.Errorf("MessageID mismatch: %s", result.MessageID)
		}
		if result.Message == "" {
			t.Error("result.Message should not be empty")
		}
	})
}

func FuzzHandlerTopicArn(f *testing.F) {
	f.Add("arn:aws:sns:ap-northeast-1:123456789012:my-topic")
	f.Add("arn:aws:sns:us-east-1:000000000000:test-topic")
	f.Add("arn:aws:sns:eu-west-1:111111111111:prod-alerts")
	f.Add("custom-topic-arn")
	f.Add("")
	f.Fuzz(func(t *testing.T, topicArn string) {
		if !utf8.ValidString(topicArn) {
			t.Skip()
		}
		mock := &mockSNS{
			publishFunc: func(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
				if aws.ToString(input.TopicArn) != topicArn {
					t.Errorf("TopicArn mismatch: want %q, got %q", topicArn, aws.ToString(input.TopicArn))
				}
				return &sns.PublishOutput{MessageId: aws.String("x")}, nil
			},
		}
		handler := Handler(mock, topicArn)
		_, err := handler(context.Background(), DeployEvent{Service: "svc", Status: "success"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

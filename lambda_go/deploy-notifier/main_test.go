package main

import (
	"context"
	"errors"
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

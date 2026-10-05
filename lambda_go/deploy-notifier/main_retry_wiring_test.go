package main

// main.go の Handler が retry.go を「実際に経由している」ことを固定する。
//
// ユーティリティを置いただけで呼び出し元に結線していない、という欠陥は
// カバレッジでは検出できない。ここでは Handler 経由で Publish を叩き、
// リトライ回数・ログフックの発火・成功時の非発火を観測する。

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// wiringRetrier は実待機ゼロ・ジッター無しのリトライ設定へ差し替える。
// 元の retrier は値型なので、復元は代入だけでよい。
func wiringRetrier(t *testing.T, maxAttempts int) {
	t.Helper()
	original := retrier
	retrier = Retrier{
		Config: RetryConfig{
			MaxAttempts: maxAttempts,
			BaseDelay:   time.Millisecond,
			MaxDelay:    time.Millisecond,
			Jitter:      false,
		},
		Sleep: func(context.Context, time.Duration) error { return nil },
	}
	t.Cleanup(func() { retrier = original })
}

func wiringEvent() DeployEvent {
	return DeployEvent{
		Service:        "my-service",
		Cluster:        "my-cluster",
		TaskDefinition: "my-task:1",
		ImageTag:       "sha-abc123",
		Status:         "success",
	}
}

func TestHandler_スロットリングならリトライして最終的に成功する(t *testing.T) {
	wiringRetrier(t, 4)

	calls := 0
	mock := &mockSNS{
		publishFunc: func(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error) {
			calls++
			if calls < 3 {
				return nil, retryTestThrottling
			}
			return &sns.PublishOutput{MessageId: aws.String("msg-retried")}, nil
		},
	}

	result, err := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:deploy-notify")(
		context.Background(), wiringEvent())
	if err != nil {
		t.Fatalf("リトライ後に成功するはず: %v", err)
	}
	if calls != 3 {
		t.Errorf("Publish の呼び出し回数 = %d, want 3（2 回リトライ）", calls)
	}
	if result.MessageID != "msg-retried" {
		t.Errorf("MessageID = %q, want %q", result.MessageID, "msg-retried")
	}
}

func TestHandler_リトライ不能なエラーは即座に返る(t *testing.T) {
	wiringRetrier(t, 4)

	calls := 0
	mock := &mockSNS{
		publishFunc: func(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error) {
			calls++
			return nil, retryTestAPIError("ValidationException", 0)
		},
	}

	if _, err := Handler(mock, "arn:aws:sns:ap-northeast-1:123456789012:deploy-notify")(
		context.Background(), wiringEvent()); err == nil {
		t.Fatal("エラーが返るはず")
	}
	if calls != 1 {
		t.Errorf("Publish の呼び出し回数 = %d, want 1（リトライしない）", calls)
	}
}

func TestHandler_リトライ時にRetryLogHookが発火する(t *testing.T) {
	wiringRetrier(t, 3)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	calls := 0
	mock := &mockSNS{
		publishFunc: func(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error) {
			calls++
			if calls < 2 {
				return nil, retryTestThrottling
			}
			return &sns.PublishOutput{MessageId: aws.String("msg-hook")}, nil
		},
	}

	if _, err := HandlerWithLogger(mock, "arn:aws:sns:ap-northeast-1:123456789012:deploy-notify", logger)(
		context.Background(), wiringEvent()); err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	var warned bool
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("ログが JSON ではない: %v (%s)", err, line)
		}
		if entry["level"] == "WARN" && entry["operation"] == RetryOperation {
			warned = true
			if entry["attempt"] == nil {
				t.Error("attempt が記録されていない")
			}
			if entry["delayMs"] == nil {
				t.Error("delayMs が記録されていない")
			}
		}
	}
	if !warned {
		t.Errorf("RetryLogHook の warn ログが出ていない: %s", buf.String())
	}
}

func TestHandler_成功時はリトライのログを出さない(t *testing.T) {
	wiringRetrier(t, 3)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	mock := &mockSNS{
		publishFunc: func(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return &sns.PublishOutput{MessageId: aws.String("msg-ok")}, nil
		},
	}

	if _, err := HandlerWithLogger(mock, "arn:aws:sns:ap-northeast-1:123456789012:deploy-notify", logger)(
		context.Background(), wiringEvent()); err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if strings.Contains(buf.String(), "リトライ") {
		t.Errorf("成功時にリトライのログが出ている: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "msg-ok") {
		t.Errorf("成功ログに messageId が含まれていない: %s", buf.String())
	}
}

# Changelog

## [0.2.0] - 2026-06-17

### Changed
- `actions/checkout` を v4 → v5 に更新（Node.js 24 対応）
- `aws-actions/configure-aws-credentials` を v4 → v5 に更新（Node.js 24 対応）
- Flask ユニットテスト 14 件・pytest ジョブを CI に追加
- Deploy ワークフローのトリガーを `push` → `workflow_dispatch`（手動実行）に変更
  - デモ用公開リポジトリのため AWS 認証情報未設定時のCI失敗を解消

## [0.1.0] - 2026-06-08

### Added
- ECR スタック（イメージスキャン・ライフサイクルルール）
- ECS スタック（VPC / ALB / Fargate / オートスケーリング / CloudWatch Logs）
- GitHub Actions CI ワークフロー（CDK build & test）
- GitHub Actions Deploy ワークフロー（ECR push → ECS deploy / OIDC 認証）
- Flask サンプルアプリ（/health・/・/info エンドポイント）
- CDK ユニットテスト（ECR: 8 件・ECS: 18 件）

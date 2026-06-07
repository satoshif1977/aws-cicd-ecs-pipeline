# Changelog

## [0.1.0] - 2026-06-08

### Added
- ECR スタック（イメージスキャン・ライフサイクルルール）
- ECS スタック（VPC / ALB / Fargate / オートスケーリング / CloudWatch Logs）
- GitHub Actions CI ワークフロー（CDK build & test）
- GitHub Actions Deploy ワークフロー（ECR push → ECS deploy / OIDC 認証）
- Flask サンプルアプリ（/health・/・/info エンドポイント）
- CDK ユニットテスト（ECR: 8 件・ECS: 18 件）

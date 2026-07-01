"""
aws-cicd-ecs-pipeline サンプルアプリ

ECS Fargate 上で稼働する Python Flask ベースの Web アプリ。
GitHub Actions → ECR → ECS デプロイのデモ用。
"""

from __future__ import annotations

import logging
import os

from flask import Flask, jsonify

# ── ロギング設定 ──────────────────────────────────────────────

logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")
logger = logging.getLogger(__name__)

# ── アプリ初期化 ──────────────────────────────────────────────

app = Flask(__name__)

PORT: int = int(os.environ.get("PORT", "8080"))
ENVIRONMENT: str = os.environ.get("ENVIRONMENT", "dev")


# ── エンドポイント ────────────────────────────────────────────


@app.route("/health")
def health() -> tuple:
    """ALB ヘルスチェックエンドポイント"""
    return jsonify({"status": "ok"}), 200


@app.route("/")
def index() -> tuple:
    """メインエンドポイント"""
    logger.info("/ へのリクエストを受信")
    return (
        jsonify(
            {
                "message": "Hello from ECS Fargate!",
                "environment": ENVIRONMENT,
            }
        ),
        200,
    )


@app.route("/info")
def info() -> tuple:
    """アプリ情報エンドポイント"""
    return (
        jsonify(
            {
                "app": "aws-cicd-ecs-pipeline",
                "runtime": "Python 3.11 / Flask",
                "infrastructure": "ECS Fargate + ALB（CDK TypeScript）",
                "ci_cd": "GitHub Actions → ECR → ECS",
                "region": os.environ.get("AWS_REGION", "ap-northeast-1"),
            }
        ),
        200,
    )


# ── エントリーポイント ────────────────────────────────────────

if __name__ == "__main__":
    logger.info("サーバー起動: port=%d / env=%s", PORT, ENVIRONMENT)
    app.run(host="0.0.0.0", port=PORT)

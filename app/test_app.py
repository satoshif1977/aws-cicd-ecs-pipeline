"""
Flask アプリ ユニットテスト

/health・/・/info エンドポイントの応答・レスポンスボディ・
環境変数反映を Flask テストクライアントで検証する。
"""

import json
from unittest.mock import patch

import pytest

from app import app as flask_app

# ── フィクスチャ ──────────────────────────────────────────────


@pytest.fixture
def client():
    flask_app.config["TESTING"] = True
    with flask_app.test_client() as c:
        yield c


# ── /health ───────────────────────────────────────────────────


class TestHealthEndpoint:
    def test_200を返す(self, client):
        response = client.get("/health")
        assert response.status_code == 200

    def test_statusがokである(self, client):
        response = client.get("/health")
        data = json.loads(response.data)
        assert data["status"] == "ok"

    def test_Content_TypeがJSON(self, client):
        response = client.get("/health")
        assert "application/json" in response.content_type


# ── / ────────────────────────────────────────────────────────


class TestIndexEndpoint:
    def test_200を返す(self, client):
        response = client.get("/")
        assert response.status_code == 200

    def test_messageがHello_from_ECS_Fargate(self, client):
        response = client.get("/")
        data = json.loads(response.data)
        assert data["message"] == "Hello from ECS Fargate!"

    def test_environmentのデフォルトがdev(self, client):
        with patch.dict("os.environ", {"ENVIRONMENT": "dev"}):
            response = client.get("/")
            data = json.loads(response.data)
            assert data["environment"] == "dev"

    def test_ENVIRONMENT環境変数がレスポンスに反映される(self, client):
        with patch("app.ENVIRONMENT", "prod"):
            response = client.get("/")
            data = json.loads(response.data)
            assert data["environment"] == "prod"


# ── /info ─────────────────────────────────────────────────────


class TestInfoEndpoint:
    def test_200を返す(self, client):
        response = client.get("/info")
        assert response.status_code == 200

    def test_必須キーが全て含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        for key in ("app", "runtime", "infrastructure", "ci_cd", "region"):
            assert key in data

    def test_appフィールドが正しい(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert data["app"] == "aws-cicd-ecs-pipeline"

    def test_runtimeにFlaskが含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert "Flask" in data["runtime"]

    def test_AWS_REGION環境変数がregionに反映される(self, client):
        with patch.dict("os.environ", {"AWS_REGION": "us-east-1"}):
            import importlib
            import app as app_module

            importlib.reload(app_module)
            c = app_module.app.test_client()
            response = c.get("/info")
            data = json.loads(response.data)
            assert data["region"] == "us-east-1"

    def test_regionのデフォルトがap_northeast_1(self, client):
        with patch.dict("os.environ", {}, clear=False):
            import os

            os.environ.pop("AWS_REGION", None)
            import importlib
            import app as app_module

            importlib.reload(app_module)
            c = app_module.app.test_client()
            response = c.get("/info")
            data = json.loads(response.data)
            assert data["region"] == "ap-northeast-1"


# ── HTTP メソッド制限 ─────────────────────────────────────────


class TestMethodNotAllowed:
    def test_healthへのPOSTは405を返す(self, client):
        response = client.post("/health")
        assert response.status_code == 405

    def test_indexへのPOSTは405を返す(self, client):
        response = client.post("/")
        assert response.status_code == 405

    def test_infoへのPOSTは405を返す(self, client):
        response = client.post("/info")
        assert response.status_code == 405

    def test_healthへのDELETEは405を返す(self, client):
        response = client.delete("/health")
        assert response.status_code == 405


# ── レスポンス Content-Type ────────────────────────────────────


class TestContentType:
    def test_indexのContent_TypeがJSON(self, client):
        response = client.get("/")
        assert "application/json" in response.content_type

    def test_infoのContent_TypeがJSON(self, client):
        response = client.get("/info")
        assert "application/json" in response.content_type


# ── /info フィールド詳細 ───────────────────────────────────────


class TestInfoEndpointDetails:
    def test_infrastructureにECSが含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert "ECS" in data["infrastructure"]

    def test_ci_cdにGitHubが含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert "GitHub" in data["ci_cd"]

    def test_runtimeにPythonが含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert "Python" in data["runtime"]


# ── レスポンスボディ詳細 ──────────────────────────────────────


class TestResponseBody:
    def test_healthのレスポンスはstatusキーのみ含む(self, client):
        response = client.get("/health")
        data = json.loads(response.data)
        assert list(data.keys()) == ["status"]

    def test_indexのレスポンスはmessageとenvironmentを含む(self, client):
        response = client.get("/")
        data = json.loads(response.data)
        assert "message" in data
        assert "environment" in data


# ── 存在しないルート ──────────────────────────────────────────


class TestNotFound:
    def test_存在しないパスは404を返す(self, client):
        response = client.get("/nonexistent")
        assert response.status_code == 404

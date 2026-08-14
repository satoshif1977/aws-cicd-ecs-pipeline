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

    def test_深いパスも404を返す(self, client):
        response = client.get("/api/v1/unknown")
        assert response.status_code == 404


# ── HTTP メソッド制限（PUT/PATCH）────────────────────────────


class TestPutPatchNotAllowed:
    def test_healthへのPUTは405を返す(self, client):
        response = client.put("/health")
        assert response.status_code == 405

    def test_indexへのPUTは405を返す(self, client):
        response = client.put("/")
        assert response.status_code == 405

    def test_infoへのPATCHは405を返す(self, client):
        response = client.patch("/info")
        assert response.status_code == 405

    def test_healthへのPATCHは405を返す(self, client):
        response = client.patch("/health")
        assert response.status_code == 405


# ── /info フィールド内容詳細 ───────────────────────────────────


class TestInfoFieldValues:
    def test_infrastructureにALBが含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert "ALB" in data["infrastructure"]

    def test_ci_cdにECRが含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert "ECR" in data["ci_cd"]

    def test_infrastructureにCDKが含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert "CDK" in data["infrastructure"]


# ── レスポンス構造検証 ────────────────────────────────────────


class TestResponseStructure:
    def test_indexのキー数は2つ(self, client):
        response = client.get("/")
        data = json.loads(response.data)
        assert len(data.keys()) == 2

    def test_infoのキー数は5つ(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert len(data.keys()) == 5

    def test_environmentはstr型(self, client):
        response = client.get("/")
        data = json.loads(response.data)
        assert isinstance(data["environment"], str)

    def test_messageはstr型(self, client):
        response = client.get("/")
        data = json.loads(response.data)
        assert isinstance(data["message"], str)


# ── /health 詳細 ──────────────────────────────────────────────


class TestHealthEndpointDetail:
    def test_statusフィールドがstr型(self, client):
        response = client.get("/health")
        data = json.loads(response.data)
        assert isinstance(data["status"], str)

    def test_HEADリクエストが200を返す(self, client):
        response = client.head("/health")
        assert response.status_code == 200

    def test_HEADレスポンスのbodyが空(self, client):
        response = client.head("/health")
        assert response.data == b""

    def test_レスポンスボディが空でない(self, client):
        response = client.get("/health")
        assert len(response.data) > 0

    def test_レスポンスがJSONデコード可能(self, client):
        response = client.get("/health")
        data = json.loads(response.data)
        assert data is not None


# ── / ENVIRONMENT バリアント ──────────────────────────────────


class TestIndexEndpointVariants:
    def test_ENVIRONMENT_prodでprodが返る(self, client):
        with patch("app.ENVIRONMENT", "prod"):
            response = client.get("/")
            data = json.loads(response.data)
            assert data["environment"] == "prod"

    def test_ENVIRONMENT_stagingでstagingが返る(self, client):
        with patch("app.ENVIRONMENT", "staging"):
            response = client.get("/")
            data = json.loads(response.data)
            assert data["environment"] == "staging"

    def test_ENVIRONMENT_testでtestが返る(self, client):
        with patch("app.ENVIRONMENT", "test"):
            response = client.get("/")
            data = json.loads(response.data)
            assert data["environment"] == "test"

    def test_environmentフィールドが空でない(self, client):
        response = client.get("/")
        data = json.loads(response.data)
        assert data["environment"] != ""

    def test_クエリパラメータ付きでも200を返す(self, client):
        response = client.get("/?debug=1")
        assert response.status_code == 200


# ── /info フィールド型検証 ─────────────────────────────────────


class TestInfoFieldTypes:
    def test_appがstr型(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert isinstance(data["app"], str)

    def test_runtimeがstr型(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert isinstance(data["runtime"], str)

    def test_infrastructureがstr型(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert isinstance(data["infrastructure"], str)

    def test_ci_cdがstr型(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert isinstance(data["ci_cd"], str)

    def test_regionがstr型(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert isinstance(data["region"], str)


# ── /info フィールド値拡張 ────────────────────────────────────


class TestInfoFieldValuesExtended:
    def test_infrastructureにFargateが含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert "Fargate" in data["infrastructure"]

    def test_runtimeに3_11が含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert "3.11" in data["runtime"]

    def test_ci_cdにECSが含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert "ECS" in data["ci_cd"]

    def test_appフィールドが空でない(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert data["app"] != ""


# ── HTTP メソッド制限（DELETE/PATCH/PUT 拡張）────────────────


class TestMethodNotAllowedExtended:
    def test_indexへのDELETEは405を返す(self, client):
        response = client.delete("/")
        assert response.status_code == 405

    def test_infoへのDELETEは405を返す(self, client):
        response = client.delete("/info")
        assert response.status_code == 405

    def test_indexへのPATCHは405を返す(self, client):
        response = client.patch("/")
        assert response.status_code == 405

    def test_infoへのPUTは405を返す(self, client):
        response = client.put("/info")
        assert response.status_code == 405


# ── 存在しないルート（拡張）──────────────────────────────────


class TestNotFoundExtended:
    def test_healthcheckパスは404を返す(self, client):
        response = client.get("/healthcheck")
        assert response.status_code == 404

    def test_apiパスは404を返す(self, client):
        response = client.get("/api")
        assert response.status_code == 404

    def test_infoのサブパスは404を返す(self, client):
        response = client.get("/info/extra")
        assert response.status_code == 404


# ── クエリパラメータ付きリクエスト ───────────────────────────


class TestQueryString:
    def test_healthへのクエリパラメータ付きGETが200を返す(self, client):
        response = client.get("/health?debug=1")
        assert response.status_code == 200

    def test_infoへのクエリパラメータ付きGETが200を返す(self, client):
        response = client.get("/info?format=json")
        assert response.status_code == 200

    def test_クエリパラメータはstatusに影響しない(self, client):
        response = client.get("/health?foo=bar")
        data = json.loads(response.data)
        assert data["status"] == "ok"


# ── HEAD リクエスト ───────────────────────────────────────────


class TestHEADRequests:
    def test_indexへのHEADが200を返す(self, client):
        response = client.head("/")
        assert response.status_code == 200

    def test_infoへのHEADが200を返す(self, client):
        response = client.head("/info")
        assert response.status_code == 200

    def test_indexへのHEADのbodyが空(self, client):
        response = client.head("/")
        assert response.data == b""


# ── / レスポンス詳細 ──────────────────────────────────────────


class TestIndexResponseDetail:
    def test_messageにFargateが含まれる(self, client):
        response = client.get("/")
        data = json.loads(response.data)
        assert "Fargate" in data["message"]

    def test_environmentはstr型(self, client):
        response = client.get("/")
        data = json.loads(response.data)
        assert isinstance(data["environment"], str)

"""
Flask アプリ 詳細・エッジケーステスト

ロギング確認・parametrize・レスポンスヘッダー詳細・
環境変数エッジケース・複数リクエスト連続実行を追加検証する。
"""

from __future__ import annotations

import json
import logging
from unittest.mock import patch

import pytest

from app import app as flask_app

# ── フィクスチャ ──────────────────────────────────────────────


@pytest.fixture
def client():
    flask_app.config["TESTING"] = True
    with flask_app.test_client() as c:
        yield c


# ── parametrize: 全エンドポイントが200を返す ─────────────────


@pytest.mark.parametrize("path", ["/health", "/", "/info"])
def test_全エンドポイントが200を返す(client, path):
    response = client.get(path)
    assert response.status_code == 200


@pytest.mark.parametrize("path", ["/health", "/", "/info"])
def test_全エンドポイントがJSONを返す(client, path):
    response = client.get(path)
    assert "application/json" in response.content_type


@pytest.mark.parametrize("path", ["/health", "/", "/info"])
def test_全エンドポイントのbodyが空でない(client, path):
    response = client.get(path)
    assert len(response.data) > 0


@pytest.mark.parametrize("path", ["/health", "/", "/info"])
def test_全エンドポイントがJSONデコード可能(client, path):
    response = client.get(path)
    data = json.loads(response.data)
    assert isinstance(data, dict)


@pytest.mark.parametrize("method", ["post", "put", "patch", "delete"])
def test_healthへの書き込みメソッドは405を返す(client, method):
    response = getattr(client, method)("/health")
    assert response.status_code == 405


@pytest.mark.parametrize("method", ["post", "put", "patch", "delete"])
def test_indexへの書き込みメソッドは405を返す(client, method):
    response = getattr(client, method)("/")
    assert response.status_code == 405


@pytest.mark.parametrize("method", ["post", "put", "patch", "delete"])
def test_infoへの書き込みメソッドは405を返す(client, method):
    response = getattr(client, method)("/info")
    assert response.status_code == 405


@pytest.mark.parametrize("path", ["/health", "/", "/info"])
def test_HEADリクエストが200を返す(client, path):
    response = client.head(path)
    assert response.status_code == 200


@pytest.mark.parametrize("path", ["/health", "/", "/info"])
def test_HEADレスポンスのbodyが空(client, path):
    response = client.head(path)
    assert response.data == b""


@pytest.mark.parametrize(
    "path",
    ["/notfound", "/api/v2/test", "/health/sub", "/info/details", "/unknown/path"],
)
def test_存在しないパスは404を返す(client, path):
    response = client.get(path)
    assert response.status_code == 404


# ── ロギングテスト ────────────────────────────────────────────


class TestLogging:
    def test_indexリクエストでINFOログが出力される(self, client, caplog):
        with caplog.at_level(logging.INFO, logger="app"):
            client.get("/")
        assert any("リクエスト" in record.message for record in caplog.records)

    def test_indexリクエストでログが1件以上出力される(self, client, caplog):
        with caplog.at_level(logging.INFO, logger="app"):
            client.get("/")
        assert len(caplog.records) >= 1

    def test_healthリクエストでログが出ないまたは例外なし(self, client, caplog):
        """health はロギングなし → 例外が起きないことを確認"""
        with caplog.at_level(logging.INFO, logger="app"):
            response = client.get("/health")
        assert response.status_code == 200


# ── レスポンスヘッダー詳細 ────────────────────────────────────


class TestResponseHeaders:
    def test_healthのContent_TypeにJSONが含まれる(self, client):
        response = client.get("/health")
        assert "json" in response.content_type.lower()

    def test_indexのContent_TypeにJSONが含まれる(self, client):
        response = client.get("/")
        assert "json" in response.content_type.lower()

    def test_infoのContent_TypeにJSONが含まれる(self, client):
        response = client.get("/info")
        assert "json" in response.content_type.lower()

    def test_healthレスポンスにContent_Lengthまたはbodyが存在する(self, client):
        response = client.get("/health")
        assert len(response.data) > 0

    def test_indexレスポンスのstatusが200(self, client):
        response = client.get("/")
        assert response.status_code == 200

    def test_infoレスポンスのstatusが200(self, client):
        response = client.get("/info")
        assert response.status_code == 200


# ── ENVIRONMENT 環境変数エッジケース ──────────────────────────


class TestEnvironmentEdgeCases:
    def test_ENVIRONMENT_qa環境でqaが返る(self, client):
        with patch("app.ENVIRONMENT", "qa"):
            response = client.get("/")
            data = json.loads(response.data)
            assert data["environment"] == "qa"

    def test_ENVIRONMENT_development環境(self, client):
        with patch("app.ENVIRONMENT", "development"):
            response = client.get("/")
            data = json.loads(response.data)
            assert data["environment"] == "development"

    def test_ENVIRONMENT_大文字PROD(self, client):
        with patch("app.ENVIRONMENT", "PROD"):
            response = client.get("/")
            data = json.loads(response.data)
            assert data["environment"] == "PROD"

    def test_ENVIRONMENT_数字を含む(self, client):
        with patch("app.ENVIRONMENT", "env-1"):
            response = client.get("/")
            data = json.loads(response.data)
            assert data["environment"] == "env-1"

    def test_ENVIRONMENT_長い文字列(self, client):
        long_env = "production-ap-northeast-1-cluster"
        with patch("app.ENVIRONMENT", long_env):
            response = client.get("/")
            data = json.loads(response.data)
            assert data["environment"] == long_env


# ── 複数リクエスト連続実行 ────────────────────────────────────


class TestMultipleRequests:
    def test_healthへの連続リクエストが全て200を返す(self, client):
        for _ in range(5):
            response = client.get("/health")
            assert response.status_code == 200

    def test_indexへの連続リクエストが全て同一messageを返す(self, client):
        results = [json.loads(client.get("/").data)["message"] for _ in range(3)]
        assert len(set(results)) == 1  # 全て同じ値

    def test_異なるエンドポイントへの連続リクエストが全て成功する(self, client):
        for path in ["/health", "/", "/info", "/health", "/"]:
            response = client.get(path)
            assert response.status_code == 200

    def test_healthとindexを交互に叩いても正常(self, client):
        for _ in range(3):
            assert client.get("/health").status_code == 200
            assert client.get("/").status_code == 200


# ── /health 追加バリエーション ────────────────────────────────


class TestHealthAdditional:
    def test_statusの値がok文字列(self, client):
        response = client.get("/health")
        data = json.loads(response.data)
        assert data["status"] == "ok"

    def test_クエリパラメータ複数でも200を返す(self, client):
        response = client.get("/health?a=1&b=2&c=3")
        assert response.status_code == 200

    def test_レスポンスボディがJSONオブジェクト(self, client):
        response = client.get("/health")
        data = json.loads(response.data)
        assert isinstance(data, dict)

    def test_レスポンスに予期しないキーが含まれない(self, client):
        response = client.get("/health")
        data = json.loads(response.data)
        unexpected = set(data.keys()) - {"status"}
        assert len(unexpected) == 0


# ── /info AWS_REGION バリエーション ──────────────────────────


class TestInfoRegionVariants:
    def test_AWS_REGION_us_west_2が反映される(self, client):
        with patch.dict("os.environ", {"AWS_REGION": "us-west-2"}):
            import importlib

            import app as app_module

            importlib.reload(app_module)
            c = app_module.app.test_client()
            response = c.get("/info")
            data = json.loads(response.data)
            assert data["region"] == "us-west-2"

    def test_AWS_REGION_eu_west_1が反映される(self, client):
        with patch.dict("os.environ", {"AWS_REGION": "eu-west-1"}):
            import importlib

            import app as app_module

            importlib.reload(app_module)
            c = app_module.app.test_client()
            response = c.get("/info")
            data = json.loads(response.data)
            assert data["region"] == "eu-west-1"

    def test_AWS_REGION_ap_southeast_1が反映される(self, client):
        with patch.dict("os.environ", {"AWS_REGION": "ap-southeast-1"}):
            import importlib

            import app as app_module

            importlib.reload(app_module)
            c = app_module.app.test_client()
            response = c.get("/info")
            data = json.loads(response.data)
            assert data["region"] == "ap-southeast-1"


# ── /info レスポンス値の詳細確認 ──────────────────────────────


class TestInfoValueDetails:
    def test_ci_cdにActionsが含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert "Actions" in data["ci_cd"]

    def test_infrastructureにTypeScriptが含まれる(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert "TypeScript" in data["infrastructure"]

    def test_ci_cdフィールドが空でない(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert len(data["ci_cd"]) > 0

    def test_infrastructureフィールドが空でない(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert len(data["infrastructure"]) > 0

    def test_runtimeフィールドが空でない(self, client):
        response = client.get("/info")
        data = json.loads(response.data)
        assert len(data["runtime"]) > 0

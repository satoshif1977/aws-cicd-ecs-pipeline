import * as cdk from 'aws-cdk-lib';
import { Match, Template } from 'aws-cdk-lib/assertions';
import { EcrStack } from '../lib/ecr-stack';

// ── フィクスチャ ─────────────────────────────────────────────

function buildTemplate(overrides: { repositoryName?: string; maxImageCount?: number } = {}): Template {
  const app = new cdk.App();
  const stack = new EcrStack(app, 'TestEcrStack', {
    repositoryName: overrides.repositoryName ?? 'test-repo',
    maxImageCount: overrides.maxImageCount,
  });
  return Template.fromStack(stack);
}

// ── ECR リポジトリ ───────────────────────────────────────────

describe('EcrStack / リポジトリ設定', () => {
  test('ECR リポジトリが 1 つ作成される', () => {
    buildTemplate().resourceCountIs('AWS::ECR::Repository', 1);
  });

  test('プッシュ時イメージスキャンが有効', () => {
    buildTemplate().hasResourceProperties('AWS::ECR::Repository', {
      ImageScanningConfiguration: { ScanOnPush: true },
    });
  });

  test('リポジトリ名が正しく設定される', () => {
    buildTemplate({ repositoryName: 'my-app' }).hasResourceProperties('AWS::ECR::Repository', {
      RepositoryName: 'my-app',
    });
  });

  test('削除ポリシーが RETAIN', () => {
    buildTemplate().hasResource('AWS::ECR::Repository', {
      DeletionPolicy: 'Retain',
    });
  });
});

// ── ライフサイクルルール ──────────────────────────────────────

describe('EcrStack / ライフサイクルルール', () => {
  test('ライフサイクルポリシーが設定される', () => {
    buildTemplate().hasResourceProperties('AWS::ECR::Repository', {
      LifecyclePolicy: Match.objectLike({
        LifecyclePolicyText: Match.anyValue(),
      }),
    });
  });

  test('デフォルトで最大 10 イメージが保持される', () => {
    const template = buildTemplate();
    const repos = template.findResources('AWS::ECR::Repository');
    const repoKeys = Object.keys(repos);
    expect(repoKeys.length).toBe(1);
    const policy = repos[repoKeys[0]].Properties.LifecyclePolicy.LifecyclePolicyText;
    const parsed = JSON.parse(policy);
    expect(parsed.rules[0].selection.countNumber).toBe(10);
  });

  test('maxImageCount を上書きできる', () => {
    const template = buildTemplate({ maxImageCount: 5 });
    const repos = template.findResources('AWS::ECR::Repository');
    const repoKeys = Object.keys(repos);
    const policy = repos[repoKeys[0]].Properties.LifecyclePolicy.LifecyclePolicyText;
    const parsed = JSON.parse(policy);
    expect(parsed.rules[0].selection.countNumber).toBe(5);
  });
});

// ── Outputs ──────────────────────────────────────────────────

describe('EcrStack / Outputs', () => {
  test('RepositoryUri が Output に存在する', () => {
    buildTemplate().hasOutput('RepositoryUri', {});
  });

  test('RepositoryName が Output に存在する', () => {
    buildTemplate().hasOutput('RepositoryName', {});
  });
});

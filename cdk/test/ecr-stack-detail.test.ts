import * as cdk from 'aws-cdk-lib';
import { Template } from 'aws-cdk-lib/assertions';
import { EcrStack } from '../lib/ecr-stack';

function buildTemplate(overrides: { repositoryName?: string; maxImageCount?: number } = {}): Template {
  const app = new cdk.App();
  const stack = new EcrStack(app, 'TestEcrStack', {
    repositoryName: overrides.repositoryName ?? 'test-repo',
    maxImageCount: overrides.maxImageCount,
  });
  return Template.fromStack(stack);
}

// ── ライフサイクルルール詳細 ──────────────────────────────────

describe('EcrStack / ライフサイクルルール詳細', () => {
  test('ライフサイクルルールの rulePriority が 1 である', () => {
    const template = buildTemplate();
    const repos = template.findResources('AWS::ECR::Repository');
    const policy = JSON.parse(Object.values(repos)[0].Properties.LifecyclePolicy.LifecyclePolicyText);
    expect(policy.rules[0].rulePriority).toBe(1);
  });

  test('ライフサイクルルールの tagStatus が any である', () => {
    const template = buildTemplate();
    const repos = template.findResources('AWS::ECR::Repository');
    const policy = JSON.parse(Object.values(repos)[0].Properties.LifecyclePolicy.LifecyclePolicyText);
    expect(policy.rules[0].selection.tagStatus).toBe('any');
  });

  test('ライフサイクルルールの countType が imageCountMoreThan である', () => {
    const template = buildTemplate();
    const repos = template.findResources('AWS::ECR::Repository');
    const policy = JSON.parse(Object.values(repos)[0].Properties.LifecyclePolicy.LifecyclePolicyText);
    expect(policy.rules[0].selection.countType).toBe('imageCountMoreThan');
  });
});

// ── Output exportName ────────────────────────────────────────

describe('EcrStack / Output exportName', () => {
  test('RepositoryUri Output に exportName が設定されている', () => {
    const template = buildTemplate();
    const outputs = template.findOutputs('RepositoryUri');
    expect(outputs['RepositoryUri'].Export).toBeDefined();
  });

  test('RepositoryName Output に exportName が設定されている', () => {
    const template = buildTemplate();
    const outputs = template.findOutputs('RepositoryName');
    expect(outputs['RepositoryName'].Export).toBeDefined();
  });

  test('RepositoryUri の exportName にスタック ID が含まれる', () => {
    const template = buildTemplate();
    const outputs = template.findOutputs('RepositoryUri');
    expect(outputs['RepositoryUri'].Export.Name).toBe('TestEcrStack-RepositoryUri');
  });

  test('RepositoryName の exportName にスタック ID が含まれる', () => {
    const template = buildTemplate();
    const outputs = template.findOutputs('RepositoryName');
    expect(outputs['RepositoryName'].Export.Name).toBe('TestEcrStack-RepositoryName');
  });
});

// ── イメージスキャン・削除ポリシー ───────────────────────────

describe('EcrStack / イメージスキャン・削除ポリシー', () => {
  test('ScanOnPush が true に設定されている', () => {
    const template = buildTemplate();
    template.hasResourceProperties('AWS::ECR::Repository', {
      ImageScanningConfiguration: { ScanOnPush: true },
    });
  });

  test('DeletionPolicy が Retain', () => {
    const template = buildTemplate();
    const repos = template.findResources('AWS::ECR::Repository');
    const repo = Object.values(repos)[0] as { DeletionPolicy: string };
    expect(repo.DeletionPolicy).toBe('Retain');
  });

  test('UpdateReplacePolicy が Retain', () => {
    const template = buildTemplate();
    const repos = template.findResources('AWS::ECR::Repository');
    const repo = Object.values(repos)[0] as { UpdateReplacePolicy: string };
    expect(repo.UpdateReplacePolicy).toBe('Retain');
  });
});

// ── リポジトリ名・リソース数 ──────────────────────────────────

describe('EcrStack / リポジトリ名・リソース数', () => {
  test('Props の repositoryName がそのままリポジトリ名に設定される', () => {
    const template = buildTemplate({ repositoryName: 'my-app-repo' });
    template.hasResourceProperties('AWS::ECR::Repository', {
      RepositoryName: 'my-app-repo',
    });
  });

  test('ECR リポジトリが 1 つだけ作成される', () => {
    const template = buildTemplate();
    template.resourceCountIs('AWS::ECR::Repository', 1);
  });
});

// ── maxImageCount カスタマイズ ────────────────────────────────

describe('EcrStack / maxImageCount', () => {
  test('maxImageCount 未指定のときデフォルト 10 になる', () => {
    const template = buildTemplate();
    const repos = template.findResources('AWS::ECR::Repository');
    const policy = JSON.parse(
      (Object.values(repos)[0] as { Properties: { LifecyclePolicy: { LifecyclePolicyText: string } } })
        .Properties.LifecyclePolicy.LifecyclePolicyText,
    );
    expect(policy.rules[0].selection.countNumber).toBe(10);
  });

  test('maxImageCount=5 のとき countNumber が 5 になる', () => {
    const template = buildTemplate({ maxImageCount: 5 });
    const repos = template.findResources('AWS::ECR::Repository');
    const policy = JSON.parse(
      (Object.values(repos)[0] as { Properties: { LifecyclePolicy: { LifecyclePolicyText: string } } })
        .Properties.LifecyclePolicy.LifecyclePolicyText,
    );
    expect(policy.rules[0].selection.countNumber).toBe(5);
  });

  test('maxImageCount=20 のとき description に 20 が含まれる', () => {
    const template = buildTemplate({ maxImageCount: 20 });
    const repos = template.findResources('AWS::ECR::Repository');
    const policy = JSON.parse(
      (Object.values(repos)[0] as { Properties: { LifecyclePolicy: { LifecyclePolicyText: string } } })
        .Properties.LifecyclePolicy.LifecyclePolicyText,
    );
    expect(policy.rules[0].description).toContain('20');
  });
});

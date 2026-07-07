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
});

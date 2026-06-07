#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { EcrStack } from '../lib/ecr-stack';
import { EcsStack } from '../lib/ecs-stack';

const app = new cdk.App();

const env = {
  account: process.env.CDK_DEFAULT_ACCOUNT,
  region: process.env.CDK_DEFAULT_REGION ?? 'ap-northeast-1',
};

const tags = {
  Project: 'aws-cicd-ecs-pipeline',
  ManagedBy: 'CDK',
  Environment: 'dev',
};

// ── ECR スタック（イメージ管理）────────────────────────────
const ecrStack = new EcrStack(app, 'CicdEcsEcrStack', {
  env,
  tags,
  repositoryName: 'cicd-ecs-app',
  maxImageCount: 10,
});

// ── ECS スタック（コンテナ実行基盤）────────────────────────
new EcsStack(app, 'CicdEcsAppStack', {
  env,
  tags,
  repository: ecrStack.repository,
  desiredCount: 2,
  cpu: 256,
  memoryLimitMiB: 512,
});

app.synth();

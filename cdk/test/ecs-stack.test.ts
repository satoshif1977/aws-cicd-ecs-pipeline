import * as cdk from 'aws-cdk-lib';
import { Match, Template } from 'aws-cdk-lib/assertions';
import { EcrStack } from '../lib/ecr-stack';
import { EcsStack } from '../lib/ecs-stack';

// ── フィクスチャ ─────────────────────────────────────────────

function buildTemplate(
  overrides: { desiredCount?: number; cpu?: number; memoryLimitMiB?: number } = {},
): Template {
  const app = new cdk.App();
  const ecrStack = new EcrStack(app, 'TestEcrStack', { repositoryName: 'test-repo' });
  const ecsStack = new EcsStack(app, 'TestEcsStack', {
    repository: ecrStack.repository,
    desiredCount: overrides.desiredCount ?? 2,
    cpu: overrides.cpu ?? 256,
    memoryLimitMiB: overrides.memoryLimitMiB ?? 512,
  });
  return Template.fromStack(ecsStack);
}

// ── VPC ──────────────────────────────────────────────────────

describe('EcsStack / VPC', () => {
  test('VPC が 1 つ作成される', () => {
    buildTemplate().resourceCountIs('AWS::EC2::VPC', 1);
  });

  test('パブリックサブネットが 2 つ存在する', () => {
    const template = buildTemplate();
    const subnets = template.findResources('AWS::EC2::Subnet');
    const publicSubnets = Object.values(subnets).filter(
      (s: any) => s.Properties?.MapPublicIpOnLaunch === true,
    );
    expect(publicSubnets.length).toBe(2);
  });

  test('NAT ゲートウェイが 1 つ作成される', () => {
    buildTemplate().resourceCountIs('AWS::EC2::NatGateway', 1);
  });
});

// ── ECS クラスター ────────────────────────────────────────────

describe('EcsStack / ECS クラスター', () => {
  test('ECS クラスターが 1 つ作成される', () => {
    buildTemplate().resourceCountIs('AWS::ECS::Cluster', 1);
  });

  test('Container Insights が有効', () => {
    buildTemplate().hasResourceProperties('AWS::ECS::Cluster', {
      ClusterSettings: [{ Name: 'containerInsights', Value: 'enabled' }],
    });
  });
});

// ── Fargate タスク定義 ────────────────────────────────────────

describe('EcsStack / タスク定義', () => {
  test('Fargate タスク定義が 1 つ作成される', () => {
    buildTemplate().resourceCountIs('AWS::ECS::TaskDefinition', 1);
  });

  test('コンテナポートが 8080 に設定される', () => {
    buildTemplate().hasResourceProperties('AWS::ECS::TaskDefinition', {
      ContainerDefinitions: Match.arrayWith([
        Match.objectLike({
          PortMappings: [{ ContainerPort: 8080, Protocol: 'tcp' }],
        }),
      ]),
    });
  });

  test('環境変数 PORT が設定される', () => {
    buildTemplate().hasResourceProperties('AWS::ECS::TaskDefinition', {
      ContainerDefinitions: Match.arrayWith([
        Match.objectLike({
          Environment: Match.arrayWith([{ Name: 'PORT', Value: '8080' }]),
        }),
      ]),
    });
  });

  test('CPU が正しく設定される', () => {
    buildTemplate({ cpu: 512 }).hasResourceProperties('AWS::ECS::TaskDefinition', {
      Cpu: '512',
    });
  });

  test('メモリが正しく設定される', () => {
    buildTemplate({ memoryLimitMiB: 1024 }).hasResourceProperties('AWS::ECS::TaskDefinition', {
      Memory: '1024',
    });
  });
});

// ── ECS サービス ──────────────────────────────────────────────

describe('EcsStack / ECS サービス', () => {
  test('ECS サービスが 1 つ作成される', () => {
    buildTemplate().resourceCountIs('AWS::ECS::Service', 1);
  });

  test('起動タイプが FARGATE', () => {
    buildTemplate().hasResourceProperties('AWS::ECS::Service', {
      LaunchType: 'FARGATE',
    });
  });
});

// ── ALB ──────────────────────────────────────────────────────

describe('EcsStack / ALB', () => {
  test('ロードバランサーが 1 つ作成される', () => {
    buildTemplate().resourceCountIs('AWS::ElasticLoadBalancingV2::LoadBalancer', 1);
  });

  test('ターゲットグループが 1 つ作成される', () => {
    buildTemplate().resourceCountIs('AWS::ElasticLoadBalancingV2::TargetGroup', 1);
  });

  test('HTTP リスナーが 1 つ作成される', () => {
    buildTemplate().resourceCountIs('AWS::ElasticLoadBalancingV2::Listener', 1);
  });
});

// ── ロギング ─────────────────────────────────────────────────

describe('EcsStack / CloudWatch Logs', () => {
  test('CloudWatch Logs グループが 1 つ作成される', () => {
    buildTemplate().resourceCountIs('AWS::Logs::LogGroup', 1);
  });

  test('ログ保持期間が 30 日', () => {
    buildTemplate().hasResourceProperties('AWS::Logs::LogGroup', {
      RetentionInDays: 30,
    });
  });
});

// ── オートスケーリング ────────────────────────────────────────

describe('EcsStack / オートスケーリング', () => {
  test('スケーラブルターゲットが設定される', () => {
    buildTemplate().resourceCountIs('AWS::ApplicationAutoScaling::ScalableTarget', 1);
  });

  test('CPU スケーリングポリシーが設定される', () => {
    buildTemplate().resourceCountIs('AWS::ApplicationAutoScaling::ScalingPolicy', 1);
  });

  test('最大タスク数がdesiredCount × 4', () => {
    buildTemplate({ desiredCount: 2 }).hasResourceProperties(
      'AWS::ApplicationAutoScaling::ScalableTarget',
      { MaxCapacity: 8 },
    );
  });
});

// ── Outputs ──────────────────────────────────────────────────

describe('EcsStack / Outputs', () => {
  test('LoadBalancerDns が Output に存在する', () => {
    buildTemplate().hasOutput('LoadBalancerDns', {});
  });

  test('ServiceName が Output に存在する', () => {
    buildTemplate().hasOutput('ServiceName', {});
  });

  test('ClusterName が Output に存在する', () => {
    buildTemplate().hasOutput('ClusterName', {});
  });
});

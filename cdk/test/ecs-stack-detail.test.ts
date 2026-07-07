import * as cdk from 'aws-cdk-lib';
import { Match, Template } from 'aws-cdk-lib/assertions';
import { EcrStack } from '../lib/ecr-stack';
import { EcsStack } from '../lib/ecs-stack';

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

// ── VPC 詳細 ─────────────────────────────────────────────────

describe('EcsStack / VPC 詳細', () => {
  test('プライベートサブネットが 2 つ存在する', () => {
    const template = buildTemplate();
    const subnets = template.findResources('AWS::EC2::Subnet');
    const privateSubnets = Object.values(subnets).filter(
      (s: any) => s.Properties?.MapPublicIpOnLaunch === false,
    );
    expect(privateSubnets.length).toBe(2);
  });

  test('Internet Gateway が 1 つ作成される', () => {
    buildTemplate().resourceCountIs('AWS::EC2::InternetGateway', 1);
  });
});

// ── ALB 詳細 ─────────────────────────────────────────────────

describe('EcsStack / ALB 詳細', () => {
  test('ALB が internet-facing で作成される', () => {
    buildTemplate().hasResourceProperties('AWS::ElasticLoadBalancingV2::LoadBalancer', {
      Scheme: 'internet-facing',
    });
  });

  test('ALB のタイプが application である', () => {
    buildTemplate().hasResourceProperties('AWS::ElasticLoadBalancingV2::LoadBalancer', {
      Type: 'application',
    });
  });

  test('ALB リスナーのポートが 80 である', () => {
    buildTemplate().hasResourceProperties('AWS::ElasticLoadBalancingV2::Listener', {
      Port: 80,
      Protocol: 'HTTP',
    });
  });
});

// ── ECS サービス詳細 ──────────────────────────────────────────

describe('EcsStack / ECS サービス詳細', () => {
  test('desiredCount が正しく設定される', () => {
    buildTemplate({ desiredCount: 3 }).hasResourceProperties('AWS::ECS::Service', {
      DesiredCount: 3,
    });
  });

  test('タスクがプライベートサブネットに配置される（パブリック IP なし）', () => {
    buildTemplate().hasResourceProperties('AWS::ECS::Service', {
      NetworkConfiguration: Match.objectLike({
        AwsvpcConfiguration: Match.objectLike({
          AssignPublicIp: 'DISABLED',
        }),
      }),
    });
  });
});

// ── CloudWatch Logs 詳細 ──────────────────────────────────────

describe('EcsStack / CloudWatch Logs 詳細', () => {
  test('ロググループ名が /ecs/ プレフィックスを持つ', () => {
    buildTemplate().hasResourceProperties('AWS::Logs::LogGroup', {
      LogGroupName: Match.stringLikeRegexp('^/ecs/'),
    });
  });

  test('ログドライバーが awslogs である', () => {
    buildTemplate().hasResourceProperties('AWS::ECS::TaskDefinition', {
      ContainerDefinitions: Match.arrayWith([
        Match.objectLike({
          LogConfiguration: Match.objectLike({
            LogDriver: 'awslogs',
          }),
        }),
      ]),
    });
  });

  test('ログストリームプレフィックスが app である', () => {
    buildTemplate().hasResourceProperties('AWS::ECS::TaskDefinition', {
      ContainerDefinitions: Match.arrayWith([
        Match.objectLike({
          LogConfiguration: Match.objectLike({
            Options: Match.objectLike({
              'awslogs-stream-prefix': 'app',
            }),
          }),
        }),
      ]),
    });
  });
});

// ── オートスケーリング詳細 ────────────────────────────────────

describe('EcsStack / オートスケーリング詳細', () => {
  test('minCapacity が desiredCount と一致する', () => {
    buildTemplate({ desiredCount: 2 }).hasResourceProperties(
      'AWS::ApplicationAutoScaling::ScalableTarget',
      { MinCapacity: 2 },
    );
  });

  test('CPU スケーリング閾値が 70% である', () => {
    buildTemplate().hasResourceProperties(
      'AWS::ApplicationAutoScaling::ScalingPolicy',
      {
        TargetTrackingScalingPolicyConfiguration: Match.objectLike({
          TargetValue: 70,
        }),
      },
    );
  });

  test('スケールインクールダウンが 60 秒である', () => {
    buildTemplate().hasResourceProperties(
      'AWS::ApplicationAutoScaling::ScalingPolicy',
      {
        TargetTrackingScalingPolicyConfiguration: Match.objectLike({
          ScaleInCooldown: 60,
        }),
      },
    );
  });
});

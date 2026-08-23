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

  test('スケールアウトクールダウンが 60 秒である', () => {
    buildTemplate().hasResourceProperties(
      'AWS::ApplicationAutoScaling::ScalingPolicy',
      {
        TargetTrackingScalingPolicyConfiguration: Match.objectLike({
          ScaleOutCooldown: 60,
        }),
      },
    );
  });

  test('desiredCount=1 のとき maxCapacity が 4 になる', () => {
    buildTemplate({ desiredCount: 1 }).hasResourceProperties(
      'AWS::ApplicationAutoScaling::ScalableTarget',
      { MaxCapacity: 4, MinCapacity: 1 },
    );
  });

  test('desiredCount=4 のとき maxCapacity が 16 になる', () => {
    buildTemplate({ desiredCount: 4 }).hasResourceProperties(
      'AWS::ApplicationAutoScaling::ScalableTarget',
      { MaxCapacity: 16, MinCapacity: 4 },
    );
  });

  test('スケーリングポリシーの PredefinedMetricType が ECSServiceAverageCPUUtilization', () => {
    buildTemplate().hasResourceProperties(
      'AWS::ApplicationAutoScaling::ScalingPolicy',
      {
        TargetTrackingScalingPolicyConfiguration: Match.objectLike({
          PredefinedMetricSpecification: Match.objectLike({
            PredefinedMetricType: 'ECSServiceAverageCPUUtilization',
          }),
        }),
      },
    );
  });
});

// ── タスク定義詳細 ──────────────────────────────────────────────

describe('EcsStack / タスク定義詳細', () => {
  test('デフォルト CPU が 256 である', () => {
    buildTemplate().hasResourceProperties('AWS::ECS::TaskDefinition', {
      Cpu: '256',
    });
  });

  test('デフォルトメモリが 512 である', () => {
    buildTemplate().hasResourceProperties('AWS::ECS::TaskDefinition', {
      Memory: '512',
    });
  });

  test('NetworkMode が awsvpc である', () => {
    buildTemplate().hasResourceProperties('AWS::ECS::TaskDefinition', {
      NetworkMode: 'awsvpc',
    });
  });

  test('RequiresCompatibilities に FARGATE が含まれる', () => {
    buildTemplate().hasResourceProperties('AWS::ECS::TaskDefinition', {
      RequiresCompatibilities: ['FARGATE'],
    });
  });

  test('cpu=1024 memoryLimitMiB=2048 のカスタム値が反映される', () => {
    const template = buildTemplate({ cpu: 1024, memoryLimitMiB: 2048 });
    template.hasResourceProperties('AWS::ECS::TaskDefinition', {
      Cpu: '1024',
      Memory: '2048',
    });
  });
});

// ── セキュリティグループ ─────────────────────────────────────────

describe('EcsStack / セキュリティグループ', () => {
  test('セキュリティグループが作成される', () => {
    const template = buildTemplate();
    const sgs = template.findResources('AWS::EC2::SecurityGroup');
    expect(Object.keys(sgs).length).toBeGreaterThanOrEqual(1);
  });
});

// ── VPC 追加 ─────────────────────────────────────────────────────

describe('EcsStack / VPC 追加', () => {
  test('サブネットが合計 4 つ（パブリック2 + プライベート2）存在する', () => {
    const template = buildTemplate();
    template.resourceCountIs('AWS::EC2::Subnet', 4);
  });

  test('ルートテーブルが作成される', () => {
    const template = buildTemplate();
    const routeTables = template.findResources('AWS::EC2::RouteTable');
    expect(Object.keys(routeTables).length).toBeGreaterThanOrEqual(1);
  });

  test('Elastic IP が NAT ゲートウェイ用に 1 つ作成される', () => {
    buildTemplate().resourceCountIs('AWS::EC2::EIP', 1);
  });
});

// ── ロググループ詳細 ─────────────────────────────────────────────

describe('EcsStack / ロググループ詳細', () => {
  test('ロググループの RemovalPolicy が Delete（DESTROY）', () => {
    const template = buildTemplate();
    const logGroups = template.findResources('AWS::Logs::LogGroup');
    const lg = Object.values(logGroups)[0] as { DeletionPolicy?: string };
    expect(lg.DeletionPolicy).toBe('Delete');
  });

  test('ロググループ名にスタック ID が含まれる', () => {
    buildTemplate().hasResourceProperties('AWS::Logs::LogGroup', {
      LogGroupName: Match.stringLikeRegexp('testecsstack'),
    });
  });
});

// ── Outputs 詳細 ─────────────────────────────────────────────────

describe('EcsStack / Outputs 詳細', () => {
  test('LoadBalancerDns の Value が定義されている', () => {
    const template = buildTemplate();
    const outputs = template.findOutputs('LoadBalancerDns');
    expect(outputs['LoadBalancerDns'].Value).toBeDefined();
  });

  test('ServiceName の Value が定義されている', () => {
    const template = buildTemplate();
    const outputs = template.findOutputs('ServiceName');
    expect(outputs['ServiceName'].Value).toBeDefined();
  });

  test('ClusterName の Value が定義されている', () => {
    const template = buildTemplate();
    const outputs = template.findOutputs('ClusterName');
    expect(outputs['ClusterName'].Value).toBeDefined();
  });
});

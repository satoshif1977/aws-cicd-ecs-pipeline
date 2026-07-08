import * as cdk from 'aws-cdk-lib';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as ecs from 'aws-cdk-lib/aws-ecs';
import * as ecsPatterns from 'aws-cdk-lib/aws-ecs-patterns';
import * as ecr from 'aws-cdk-lib/aws-ecr';
import * as logs from 'aws-cdk-lib/aws-logs';
import { Construct } from 'constructs';
import { NagSuppressions } from 'cdk-nag';

// ── Props ────────────────────────────────────────────────────

export interface EcsStackProps extends cdk.StackProps {
  /** コンテナイメージを取得する ECR リポジトリ */
  readonly repository: ecr.IRepository;
  /** デプロイするイメージタグ（デフォルト: latest） */
  readonly imageTag?: string;
  /** 起動タスク数（デフォルト: 2） */
  readonly desiredCount?: number;
  /** タスク CPU ユニット（デフォルト: 256） */
  readonly cpu?: number;
  /** タスクメモリ MiB（デフォルト: 512） */
  readonly memoryLimitMiB?: number;
}

// ── Stack ────────────────────────────────────────────────────

/**
 * ECS スタック
 *
 * - VPC（パブリック + プライベートサブネット / 2AZ）
 * - ECS クラスター（Container Insights 有効）
 * - ALB + Fargate サービス（プライベートサブネット配置）
 * - CloudWatch Logs（30日保持）
 * - CPU オートスケーリング（70% 閾値）
 */
export class EcsStack extends cdk.Stack {
  public readonly service: ecsPatterns.ApplicationLoadBalancedFargateService;
  public readonly cluster: ecs.Cluster;

  constructor(scope: Construct, id: string, props: EcsStackProps) {
    super(scope, id, props);

    const imageTag = props.imageTag ?? 'latest';
    const desiredCount = props.desiredCount ?? 2;
    const cpu = props.cpu ?? 256;
    const memoryLimitMiB = props.memoryLimitMiB ?? 512;

    // ── VPC ──────────────────────────────────────────────────
    const vpc = new ec2.Vpc(this, 'Vpc', {
      maxAzs: 2,
      natGateways: 1,
      subnetConfiguration: [
        {
          name: 'public',
          subnetType: ec2.SubnetType.PUBLIC,
          cidrMask: 24,
        },
        {
          name: 'private',
          subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS,
          cidrMask: 24,
        },
      ],
    });

    // ── ECS クラスター ───────────────────────────────────────
    this.cluster = new ecs.Cluster(this, 'Cluster', {
      vpc,
      containerInsightsV2: ecs.ContainerInsights.ENABLED,
    });

    // ── CloudWatch Logs ──────────────────────────────────────
    const logGroup = new logs.LogGroup(this, 'LogGroup', {
      logGroupName: `/ecs/${id.toLowerCase()}-app`,
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // ── ALB + Fargate サービス ───────────────────────────────
    this.service = new ecsPatterns.ApplicationLoadBalancedFargateService(
      this,
      'FargateService',
      {
        cluster: this.cluster,
        desiredCount,
        cpu,
        memoryLimitMiB,
        publicLoadBalancer: true,
        assignPublicIp: false,
        taskSubnets: {
          subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS,
        },
        taskImageOptions: {
          image: ecs.ContainerImage.fromEcrRepository(props.repository, imageTag),
          containerPort: 8080,
          logDriver: ecs.LogDrivers.awsLogs({
            logGroup,
            streamPrefix: 'app',
          }),
          environment: {
            PORT: '8080',
          },
        },
      },
    );

    // ── オートスケーリング ───────────────────────────────────
    const scaling = this.service.service.autoScaleTaskCount({
      minCapacity: desiredCount,
      maxCapacity: desiredCount * 4,
    });

    scaling.scaleOnCpuUtilization('CpuScaling', {
      targetUtilizationPercent: 70,
      scaleInCooldown: cdk.Duration.seconds(60),
      scaleOutCooldown: cdk.Duration.seconds(60),
    });

    // ── Outputs ──────────────────────────────────────────────
    new cdk.CfnOutput(this, 'LoadBalancerDns', {
      value: this.service.loadBalancer.loadBalancerDnsName,
      description: 'ALB DNS 名',
    });

    new cdk.CfnOutput(this, 'ServiceName', {
      value: this.service.service.serviceName,
      description: 'ECS サービス名',
    });

    new cdk.CfnOutput(this, 'ClusterName', {
      value: this.cluster.clusterName,
      description: 'ECS クラスター名',
    });

    // ── cdk-nag suppressions（dev 環境の意図的な省略） ────────────
    NagSuppressions.addStackSuppressions(this, [
      {
        id: 'AwsSolutions-VPC7',
        reason: 'dev 環境のため VPC Flow Logs は省略。本番では CloudWatch Logs への Flow Log を有効化すること。',
      },
      {
        id: 'AwsSolutions-ELB2',
        reason: 'dev 環境のため ALB アクセスログは省略。本番では S3 バケットへのアクセスログを有効化すること。',
      },
      {
        id: 'AwsSolutions-EC23',
        reason: 'ALB の HTTP(80) はインターネット公開が目的のため 0.0.0.0/0 を許可。意図的な設定。',
      },
      {
        id: 'AwsSolutions-ECS2',
        reason: 'PORT 環境変数はポート番号のみで機密情報を含まない。本番では Secrets Manager / SSM Parameter Store への移行を検討すること。',
      },
      {
        id: 'AwsSolutions-IAM5',
        reason: 'CDK ApplicationLoadBalancedFargateService が自動生成する ECS タスク実行ロールのワイルドカード権限。ECR イメージ取得・CloudWatch Logs 書き込みに必要な標準パターン。',
      },
    ]);
  }
}

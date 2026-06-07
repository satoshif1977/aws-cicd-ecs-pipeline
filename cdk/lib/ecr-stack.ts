import * as cdk from 'aws-cdk-lib';
import * as ecr from 'aws-cdk-lib/aws-ecr';
import { Construct } from 'constructs';

// ── Props ────────────────────────────────────────────────────

export interface EcrStackProps extends cdk.StackProps {
  /** ECR リポジトリ名 */
  readonly repositoryName: string;
  /** 保持するイメージの最大数（デフォルト: 10） */
  readonly maxImageCount?: number;
}

// ── Stack ────────────────────────────────────────────────────

/**
 * ECR スタック
 *
 * - プッシュ時イメージスキャン有効
 * - ライフサイクルルール（古いイメージを自動削除）
 * - リポジトリ URI を Output に出力
 */
export class EcrStack extends cdk.Stack {
  /** 他スタックから参照できる ECR リポジトリ */
  public readonly repository: ecr.Repository;

  constructor(scope: Construct, id: string, props: EcrStackProps) {
    super(scope, id, props);

    const maxImageCount = props.maxImageCount ?? 10;

    // ── ECR リポジトリ ───────────────────────────────────────
    this.repository = new ecr.Repository(this, 'Repository', {
      repositoryName: props.repositoryName,
      imageScanOnPush: true,
      removalPolicy: cdk.RemovalPolicy.RETAIN,
      lifecycleRules: [
        {
          rulePriority: 1,
          description: `最新 ${maxImageCount} イメージのみ保持`,
          maxImageCount,
          tagStatus: ecr.TagStatus.ANY,
        },
      ],
    });

    // ── Outputs ──────────────────────────────────────────────
    new cdk.CfnOutput(this, 'RepositoryUri', {
      value: this.repository.repositoryUri,
      description: 'ECR リポジトリ URI',
      exportName: `${id}-RepositoryUri`,
    });

    new cdk.CfnOutput(this, 'RepositoryName', {
      value: this.repository.repositoryName,
      description: 'ECR リポジトリ名',
      exportName: `${id}-RepositoryName`,
    });
  }
}

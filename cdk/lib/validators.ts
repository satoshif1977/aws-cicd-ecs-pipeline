/**
 * ECS/Fargate デプロイ前設定バリデーター
 *
 * Fargate の CPU/Memory 組み合わせ制約や ECS サービス設定の
 * 妥当性をデプロイ前に検証する純粋関数群。
 *
 * Fargate の CPU/Memory 制約は AWS 公式ドキュメントに基づく:
 * https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-cpu-memory-error.html
 */

// ── 型定義 ────────────────────────────────────────────────────

export interface ValidationError {
  field: string;
  message: string;
  severity: "error" | "warning";
}

export interface FargateTaskConfig {
  cpu: number;
  memoryLimitMiB: number;
  desiredCount: number;
  containerPort: number;
  imageTag: string;
}

export interface EcrConfig {
  repositoryName: string;
  maxImageCount: number;
}

export interface AutoScalingConfig {
  minCapacity: number;
  maxCapacity: number;
  targetCpuUtilization: number;
}

// ── Fargate CPU/Memory 組み合わせ制約 ─────────────────────────

/**
 * Fargate がサポートする CPU/Memory の有効な組み合わせ
 * キー: CPU ユニット、値: 許可される Memory MiB の配列
 */
export const FARGATE_CPU_MEMORY_MAP: ReadonlyMap<number, readonly number[]> = new Map([
  [256,  [512, 1024, 2048]],
  [512,  [1024, 2048, 3072, 4096]],
  [1024, [2048, 3072, 4096, 5120, 6144, 7168, 8192]],
  [2048, [4096, 5120, 6144, 7168, 8192, 9216, 10240, 11264, 12288, 13312, 14336, 15360, 16384]],
  [4096, [8192, 9216, 10240, 11264, 12288, 13312, 14336, 15360, 16384,
          17408, 18432, 19456, 20480, 21504, 22528, 23552, 24576, 25600, 26624, 27648, 28672, 29696, 30720]],
]);

/** サポートされる Fargate CPU 値 */
export const VALID_CPU_VALUES = [...FARGATE_CPU_MEMORY_MAP.keys()] as const;

/** 有効なコンテナポート範囲 */
export const MIN_PORT = 1;
export const MAX_PORT = 65535;

/** ECR リポジトリ名の正規表現パターン */
export const ECR_REPO_NAME_PATTERN = /^[a-z][a-z0-9._/-]{1,255}$/;

// ── Fargate CPU/Memory バリデーション ──────────────────────────

/** CPU 値が Fargate でサポートされているか */
export function isValidFargateCpu(cpu: number): boolean {
  return FARGATE_CPU_MEMORY_MAP.has(cpu);
}

/** CPU/Memory 組み合わせが Fargate でサポートされているか */
export function isValidCpuMemoryCombo(cpu: number, memoryMiB: number): boolean {
  const allowedMemory = FARGATE_CPU_MEMORY_MAP.get(cpu);
  if (!allowedMemory) return false;
  return allowedMemory.includes(memoryMiB);
}

/** 指定 CPU で許可される Memory 値の一覧を返す（無効な CPU は空配列） */
export function getAllowedMemoryForCpu(cpu: number): readonly number[] {
  return FARGATE_CPU_MEMORY_MAP.get(cpu) ?? [];
}

/** Fargate タスクの CPU/Memory を検証する */
export function validateCpuMemory(cpu: number, memoryMiB: number): ValidationError[] {
  const errors: ValidationError[] = [];

  if (!isValidFargateCpu(cpu)) {
    errors.push({
      field: "cpu",
      message: `Fargate でサポートされていない CPU 値: ${cpu}。有効値: ${VALID_CPU_VALUES.join(", ")}`,
      severity: "error",
    });
    return errors;
  }

  if (!isValidCpuMemoryCombo(cpu, memoryMiB)) {
    const allowed = getAllowedMemoryForCpu(cpu);
    errors.push({
      field: "memoryLimitMiB",
      message: `CPU ${cpu} で使用できない Memory: ${memoryMiB} MiB。有効値: ${allowed.join(", ")}`,
      severity: "error",
    });
  }

  return errors;
}

// ── コンテナ設定バリデーション ─────────────────────────────────

/** コンテナポートが有効範囲内か */
export function isValidPort(port: number): boolean {
  return Number.isInteger(port) && port >= MIN_PORT && port <= MAX_PORT;
}

/** イメージタグが空でないか */
export function isValidImageTag(tag: string): boolean {
  return tag.trim().length > 0 && tag.length <= 128;
}

/** desiredCount が妥当な範囲か */
export function isValidDesiredCount(count: number): boolean {
  return Number.isInteger(count) && count >= 0 && count <= 10;
}

/** Fargate タスク設定を検証する */
export function validateTaskConfig(config: FargateTaskConfig): ValidationError[] {
  const errors: ValidationError[] = [];

  errors.push(...validateCpuMemory(config.cpu, config.memoryLimitMiB));

  if (!isValidPort(config.containerPort)) {
    errors.push({
      field: "containerPort",
      message: `無効なコンテナポート: ${config.containerPort}（有効範囲: ${MIN_PORT}〜${MAX_PORT}）`,
      severity: "error",
    });
  }

  if (!isValidImageTag(config.imageTag)) {
    errors.push({
      field: "imageTag",
      message: "イメージタグは 1〜128 文字の空でない文字列にしてください",
      severity: "error",
    });
  }

  if (!isValidDesiredCount(config.desiredCount)) {
    errors.push({
      field: "desiredCount",
      message: `desiredCount は 0〜10 の整数にしてください（現在: ${config.desiredCount}）`,
      severity: "error",
    });
  }

  if (config.desiredCount === 0) {
    errors.push({
      field: "desiredCount",
      message: "desiredCount が 0 です。サービスにタスクが起動しません",
      severity: "warning",
    });
  }

  return errors;
}

// ── ECR バリデーション ────────────────────────────────────────

/** ECR リポジトリ名が命名規則に沿っているか */
export function isValidRepositoryName(name: string): boolean {
  return ECR_REPO_NAME_PATTERN.test(name);
}

/** ECR 設定を検証する */
export function validateEcrConfig(config: EcrConfig): ValidationError[] {
  const errors: ValidationError[] = [];

  if (!isValidRepositoryName(config.repositoryName)) {
    errors.push({
      field: "repositoryName",
      message: `無効な ECR リポジトリ名: "${config.repositoryName}"。小文字英数字・ハイフン・スラッシュ・ドット・アンダースコアのみ使用可`,
      severity: "error",
    });
  }

  if (!Number.isInteger(config.maxImageCount) || config.maxImageCount < 1) {
    errors.push({
      field: "maxImageCount",
      message: `maxImageCount は 1 以上の整数にしてください（現在: ${config.maxImageCount}）`,
      severity: "error",
    });
  }

  if (config.maxImageCount > 1000) {
    errors.push({
      field: "maxImageCount",
      message: `maxImageCount が ${config.maxImageCount} と大きすぎます。コスト最適化のため 100 以下を推奨`,
      severity: "warning",
    });
  }

  return errors;
}

// ── オートスケーリングバリデーション ──────────────────────────

/** オートスケーリング設定を検証する */
export function validateAutoScaling(config: AutoScalingConfig): ValidationError[] {
  const errors: ValidationError[] = [];

  if (config.minCapacity < 0) {
    errors.push({ field: "minCapacity", message: "minCapacity は 0 以上にしてください", severity: "error" });
  }

  if (config.maxCapacity < 1) {
    errors.push({ field: "maxCapacity", message: "maxCapacity は 1 以上にしてください", severity: "error" });
  }

  if (config.minCapacity > config.maxCapacity) {
    errors.push({
      field: "minCapacity",
      message: `minCapacity（${config.minCapacity}）が maxCapacity（${config.maxCapacity}）を超えています`,
      severity: "error",
    });
  }

  if (config.targetCpuUtilization <= 0 || config.targetCpuUtilization > 100) {
    errors.push({
      field: "targetCpuUtilization",
      message: `targetCpuUtilization は 1〜100 の範囲にしてください（現在: ${config.targetCpuUtilization}）`,
      severity: "error",
    });
  }

  if (config.targetCpuUtilization > 90) {
    errors.push({
      field: "targetCpuUtilization",
      message: "targetCpuUtilization が 90% を超えています。スケールアウトが間に合わない可能性があります",
      severity: "warning",
    });
  }

  return errors;
}

// ── ユーティリティ ────────────────────────────────────────────

/** エラーの有無を判定する（warning は含まない） */
export function hasErrors(errors: ValidationError[]): boolean {
  return errors.some((e) => e.severity === "error");
}

/** エラーをフォーマットする */
export function formatErrors(errors: ValidationError[]): string {
  if (errors.length === 0) return "すべてのチェックが通過しました";
  return errors
    .map((e) => `[${e.severity.toUpperCase()}] ${e.field}: ${e.message}`)
    .join("\n");
}

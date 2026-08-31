"use strict";

import {
  isValidFargateCpu,
  isValidCpuMemoryCombo,
  getAllowedMemoryForCpu,
  validateCpuMemory,
  isValidPort,
  isValidImageTag,
  isValidDesiredCount,
  validateTaskConfig,
  isValidRepositoryName,
  validateEcrConfig,
  validateAutoScaling,
  hasErrors,
  formatErrors,
  FARGATE_CPU_MEMORY_MAP,
  VALID_CPU_VALUES,
  MIN_PORT,
  MAX_PORT,
} from "../lib/validators";
import type { FargateTaskConfig, ValidationError } from "../lib/validators";

// ── テスト用ヘルパー ─────────────────────────────────────────

const validTaskConfig: FargateTaskConfig = {
  cpu: 256,
  memoryLimitMiB: 512,
  desiredCount: 2,
  containerPort: 8080,
  imageTag: "latest",
};

// ── 定数テスト ────────────────────────────────────────────────

describe("constants", () => {
  test("FARGATE_CPU_MEMORY_MAP は 5 つの CPU 値を持つ", () => {
    expect(FARGATE_CPU_MEMORY_MAP.size).toBe(5);
  });

  test("VALID_CPU_VALUES は 256, 512, 1024, 2048, 4096", () => {
    expect(VALID_CPU_VALUES).toEqual([256, 512, 1024, 2048, 4096]);
  });

  test("CPU 256 は 3 つの Memory を許可", () => {
    expect(FARGATE_CPU_MEMORY_MAP.get(256)).toEqual([512, 1024, 2048]);
  });

  test("CPU 512 は 4 つの Memory を許可", () => {
    expect(FARGATE_CPU_MEMORY_MAP.get(512)).toHaveLength(4);
  });

  test("CPU 1024 は 7 つの Memory を許可", () => {
    expect(FARGATE_CPU_MEMORY_MAP.get(1024)).toHaveLength(7);
  });

  test("ポート範囲は 1〜65535", () => {
    expect(MIN_PORT).toBe(1);
    expect(MAX_PORT).toBe(65535);
  });
});

// ── isValidFargateCpu ────────────────────────────────────────

describe("isValidFargateCpu", () => {
  test.each([256, 512, 1024, 2048, 4096])("CPU %d は有効", (cpu) => {
    expect(isValidFargateCpu(cpu)).toBe(true);
  });

  test.each([0, 128, 768, 3072, 8192])("CPU %d は無効", (cpu) => {
    expect(isValidFargateCpu(cpu)).toBe(false);
  });

  test("負の値は無効", () => {
    expect(isValidFargateCpu(-256)).toBe(false);
  });
});

// ── isValidCpuMemoryCombo ────────────────────────────────────

describe("isValidCpuMemoryCombo", () => {
  test("CPU 256 / Memory 512 は有効", () => {
    expect(isValidCpuMemoryCombo(256, 512)).toBe(true);
  });

  test("CPU 256 / Memory 1024 は有効", () => {
    expect(isValidCpuMemoryCombo(256, 1024)).toBe(true);
  });

  test("CPU 256 / Memory 2048 は有効", () => {
    expect(isValidCpuMemoryCombo(256, 2048)).toBe(true);
  });

  test("CPU 256 / Memory 4096 は無効", () => {
    expect(isValidCpuMemoryCombo(256, 4096)).toBe(false);
  });

  test("CPU 512 / Memory 512 は無効", () => {
    expect(isValidCpuMemoryCombo(512, 512)).toBe(false);
  });

  test("CPU 512 / Memory 1024 は有効", () => {
    expect(isValidCpuMemoryCombo(512, 1024)).toBe(true);
  });

  test("CPU 1024 / Memory 2048 は有効", () => {
    expect(isValidCpuMemoryCombo(1024, 2048)).toBe(true);
  });

  test("CPU 1024 / Memory 8192 は有効", () => {
    expect(isValidCpuMemoryCombo(1024, 8192)).toBe(true);
  });

  test("CPU 2048 / Memory 4096 は有効", () => {
    expect(isValidCpuMemoryCombo(2048, 4096)).toBe(true);
  });

  test("CPU 4096 / Memory 30720 は有効", () => {
    expect(isValidCpuMemoryCombo(4096, 30720)).toBe(true);
  });

  test("無効な CPU は全て false", () => {
    expect(isValidCpuMemoryCombo(128, 512)).toBe(false);
  });
});

// ── getAllowedMemoryForCpu ────────────────────────────────────

describe("getAllowedMemoryForCpu", () => {
  test("CPU 256 は [512, 1024, 2048]", () => {
    expect(getAllowedMemoryForCpu(256)).toEqual([512, 1024, 2048]);
  });

  test("無効な CPU は空配列", () => {
    expect(getAllowedMemoryForCpu(128)).toEqual([]);
  });

  test("CPU 4096 は最大数の Memory を持つ", () => {
    const memory = getAllowedMemoryForCpu(4096);
    expect(memory.length).toBeGreaterThan(10);
    expect(memory[0]).toBe(8192);
    expect(memory[memory.length - 1]).toBe(30720);
  });
});

// ── validateCpuMemory ────────────────────────────────────────

describe("validateCpuMemory", () => {
  test("有効な組み合わせはエラーなし", () => {
    expect(validateCpuMemory(256, 512)).toHaveLength(0);
  });

  test("無効な CPU は error", () => {
    const errors = validateCpuMemory(128, 512);
    expect(errors).toHaveLength(1);
    expect(errors[0].field).toBe("cpu");
  });

  test("有効な CPU + 無効な Memory は error", () => {
    const errors = validateCpuMemory(256, 4096);
    expect(errors).toHaveLength(1);
    expect(errors[0].field).toBe("memoryLimitMiB");
  });

  test("エラーメッセージに有効値が含まれる", () => {
    const errors = validateCpuMemory(256, 4096);
    expect(errors[0].message).toContain("512");
  });
});

// ── isValidPort ──────────────────────────────────────────────

describe("isValidPort", () => {
  test("8080 は有効", () => {
    expect(isValidPort(8080)).toBe(true);
  });

  test("80 は有効", () => {
    expect(isValidPort(80)).toBe(true);
  });

  test("1 は有効（境界値）", () => {
    expect(isValidPort(1)).toBe(true);
  });

  test("65535 は有効（境界値）", () => {
    expect(isValidPort(65535)).toBe(true);
  });

  test("0 は無効", () => {
    expect(isValidPort(0)).toBe(false);
  });

  test("65536 は無効", () => {
    expect(isValidPort(65536)).toBe(false);
  });

  test("負の値は無効", () => {
    expect(isValidPort(-1)).toBe(false);
  });

  test("小数は無効", () => {
    expect(isValidPort(80.5)).toBe(false);
  });
});

// ── isValidImageTag ──────────────────────────────────────────

describe("isValidImageTag", () => {
  test("latest は有効", () => {
    expect(isValidImageTag("latest")).toBe(true);
  });

  test("v1.0.0 は有効", () => {
    expect(isValidImageTag("v1.0.0")).toBe(true);
  });

  test("空文字は無効", () => {
    expect(isValidImageTag("")).toBe(false);
  });

  test("スペースのみは無効", () => {
    expect(isValidImageTag("   ")).toBe(false);
  });

  test("129文字は無効", () => {
    expect(isValidImageTag("a".repeat(129))).toBe(false);
  });

  test("128文字は有効（境界値）", () => {
    expect(isValidImageTag("a".repeat(128))).toBe(true);
  });
});

// ── isValidDesiredCount ──────────────────────────────────────

describe("isValidDesiredCount", () => {
  test("2 は有効", () => {
    expect(isValidDesiredCount(2)).toBe(true);
  });

  test("0 は有効（スケールダウン用）", () => {
    expect(isValidDesiredCount(0)).toBe(true);
  });

  test("10 は有効（境界値）", () => {
    expect(isValidDesiredCount(10)).toBe(true);
  });

  test("11 は無効", () => {
    expect(isValidDesiredCount(11)).toBe(false);
  });

  test("-1 は無効", () => {
    expect(isValidDesiredCount(-1)).toBe(false);
  });

  test("小数は無効", () => {
    expect(isValidDesiredCount(1.5)).toBe(false);
  });
});

// ── validateTaskConfig ───────────────────────────────────────

describe("validateTaskConfig", () => {
  test("有効な設定はエラーなし", () => {
    expect(validateTaskConfig(validTaskConfig)).toHaveLength(0);
  });

  test("無効な CPU/Memory でエラー", () => {
    const config = { ...validTaskConfig, cpu: 128 };
    expect(hasErrors(validateTaskConfig(config))).toBe(true);
  });

  test("無効なポートでエラー", () => {
    const config = { ...validTaskConfig, containerPort: 0 };
    const errors = validateTaskConfig(config);
    expect(errors.some((e) => e.field === "containerPort")).toBe(true);
  });

  test("空のイメージタグでエラー", () => {
    const config = { ...validTaskConfig, imageTag: "" };
    const errors = validateTaskConfig(config);
    expect(errors.some((e) => e.field === "imageTag")).toBe(true);
  });

  test("desiredCount=0 は warning（error ではない）", () => {
    const config = { ...validTaskConfig, desiredCount: 0 };
    const errors = validateTaskConfig(config);
    const countErrors = errors.filter((e) => e.field === "desiredCount");
    expect(countErrors.some((e) => e.severity === "warning")).toBe(true);
    expect(countErrors.every((e) => e.severity !== "error")).toBe(true);
  });

  test("desiredCount=11 は error", () => {
    const config = { ...validTaskConfig, desiredCount: 11 };
    const errors = validateTaskConfig(config);
    expect(errors.some((e) => e.field === "desiredCount" && e.severity === "error")).toBe(true);
  });

  test("複数エラーを同時に検出", () => {
    const config: FargateTaskConfig = {
      cpu: 128,
      memoryLimitMiB: 256,
      desiredCount: -1,
      containerPort: 0,
      imageTag: "",
    };
    const errors = validateTaskConfig(config);
    expect(errors.length).toBeGreaterThanOrEqual(3);
  });
});

// ── isValidRepositoryName ────────────────────────────────────

describe("isValidRepositoryName", () => {
  test("cicd-ecs-app は有効", () => {
    expect(isValidRepositoryName("cicd-ecs-app")).toBe(true);
  });

  test("my-org/my-app は有効", () => {
    expect(isValidRepositoryName("my-org/my-app")).toBe(true);
  });

  test("大文字を含むと無効", () => {
    expect(isValidRepositoryName("MyApp")).toBe(false);
  });

  test("空文字は無効", () => {
    expect(isValidRepositoryName("")).toBe(false);
  });

  test("数字始まりは無効", () => {
    expect(isValidRepositoryName("1-app")).toBe(false);
  });

  test("ドット付き名前は有効", () => {
    expect(isValidRepositoryName("my.app")).toBe(true);
  });

  test("アンダースコア付き名前は有効", () => {
    expect(isValidRepositoryName("my_app")).toBe(true);
  });
});

// ── validateEcrConfig ────────────────────────────────────────

describe("validateEcrConfig", () => {
  test("有効な設定はエラーなし", () => {
    expect(validateEcrConfig({ repositoryName: "cicd-ecs-app", maxImageCount: 10 })).toHaveLength(0);
  });

  test("無効なリポジトリ名は error", () => {
    const errors = validateEcrConfig({ repositoryName: "INVALID", maxImageCount: 10 });
    expect(errors.some((e) => e.field === "repositoryName")).toBe(true);
  });

  test("maxImageCount=0 は error", () => {
    const errors = validateEcrConfig({ repositoryName: "app", maxImageCount: 0 });
    expect(errors.some((e) => e.field === "maxImageCount" && e.severity === "error")).toBe(true);
  });

  test("maxImageCount=1001 は warning", () => {
    const errors = validateEcrConfig({ repositoryName: "app", maxImageCount: 1001 });
    expect(errors.some((e) => e.field === "maxImageCount" && e.severity === "warning")).toBe(true);
  });

  test("maxImageCount=1 は有効", () => {
    expect(validateEcrConfig({ repositoryName: "app", maxImageCount: 1 })).toHaveLength(0);
  });
});

// ── validateAutoScaling ──────────────────────────────────────

describe("validateAutoScaling", () => {
  test("有効な設定はエラーなし", () => {
    expect(validateAutoScaling({ minCapacity: 2, maxCapacity: 8, targetCpuUtilization: 70 })).toHaveLength(0);
  });

  test("minCapacity < 0 は error", () => {
    const errors = validateAutoScaling({ minCapacity: -1, maxCapacity: 4, targetCpuUtilization: 70 });
    expect(errors.some((e) => e.field === "minCapacity")).toBe(true);
  });

  test("maxCapacity < 1 は error", () => {
    const errors = validateAutoScaling({ minCapacity: 0, maxCapacity: 0, targetCpuUtilization: 70 });
    expect(errors.some((e) => e.field === "maxCapacity")).toBe(true);
  });

  test("minCapacity > maxCapacity は error", () => {
    const errors = validateAutoScaling({ minCapacity: 5, maxCapacity: 3, targetCpuUtilization: 70 });
    expect(errors.some((e) => e.message.includes("超えています"))).toBe(true);
  });

  test("targetCpuUtilization=0 は error", () => {
    const errors = validateAutoScaling({ minCapacity: 1, maxCapacity: 4, targetCpuUtilization: 0 });
    expect(errors.some((e) => e.field === "targetCpuUtilization" && e.severity === "error")).toBe(true);
  });

  test("targetCpuUtilization=101 は error", () => {
    const errors = validateAutoScaling({ minCapacity: 1, maxCapacity: 4, targetCpuUtilization: 101 });
    expect(errors.some((e) => e.field === "targetCpuUtilization" && e.severity === "error")).toBe(true);
  });

  test("targetCpuUtilization=95 は warning", () => {
    const errors = validateAutoScaling({ minCapacity: 1, maxCapacity: 4, targetCpuUtilization: 95 });
    const cpuErr = errors.find((e) => e.field === "targetCpuUtilization" && e.severity === "warning");
    expect(cpuErr).toBeDefined();
  });

  test("targetCpuUtilization=90 は warning なし（ちょうど境界）", () => {
    const errors = validateAutoScaling({ minCapacity: 1, maxCapacity: 4, targetCpuUtilization: 90 });
    expect(errors.some((e) => e.severity === "warning")).toBe(false);
  });
});

// ── hasErrors ────────────────────────────────────────────────

describe("hasErrors", () => {
  test("空配列は false", () => {
    expect(hasErrors([])).toBe(false);
  });

  test("error があれば true", () => {
    const errors: ValidationError[] = [{ field: "f", message: "m", severity: "error" }];
    expect(hasErrors(errors)).toBe(true);
  });

  test("warning のみは false", () => {
    const errors: ValidationError[] = [{ field: "f", message: "m", severity: "warning" }];
    expect(hasErrors(errors)).toBe(false);
  });
});

// ── formatErrors ─────────────────────────────────────────────

describe("formatErrors", () => {
  test("空配列は成功メッセージ", () => {
    expect(formatErrors([])).toContain("すべてのチェックが通過");
  });

  test("[ERROR] プレフィックスを含む", () => {
    const errors: ValidationError[] = [{ field: "cpu", message: "無効", severity: "error" }];
    expect(formatErrors(errors)).toContain("[ERROR]");
  });

  test("[WARNING] プレフィックスを含む", () => {
    const errors: ValidationError[] = [{ field: "nat", message: "少ない", severity: "warning" }];
    expect(formatErrors(errors)).toContain("[WARNING]");
  });

  test("複数エラーは改行で結合", () => {
    const errors: ValidationError[] = [
      { field: "f1", message: "m1", severity: "error" },
      { field: "f2", message: "m2", severity: "warning" },
    ];
    expect(formatErrors(errors).split("\n")).toHaveLength(2);
  });
});

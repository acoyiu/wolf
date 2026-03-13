# Spec: docker-deployment

## Step 1: Context

**Depends on:** 所有其他 features（打包整合）
**Depended by:** none
**Related CONTEXT.md:**
- CONTEXT.md (project-level)

**Technical constraints:**
- Multi-stage Docker build：Go 編譯 + Vue 建置 → Alpine 執行映像
- 最終映像 < 30MB
- 單一埠（如 3000）同時服務 WebSocket 和前端靜態檔案
- 支援環境變數配置（埠號、計時器設定等）
- 可部署至 Kubernetes（提供 Deployment + Service YAML）

**Scope:**
- Dockerfile（multi-stage）
- k8s Deployment + Service YAML
- 健康檢查端點（/healthz）
- 啟動配置（環境變數）

## Step 2: Intent (Human fills in)

Context is totally correct as intent.

## Step 3: Examples (Agent generates → Human approves)

| #   | Input | Output | Note               |
| --- | ----- | ------ | ------------------ |
| 1   |       |        | happy path         |
| 2   |       |        | happy path variant |
| 3   |       |        | edge case          |
| 4   |       |        | edge case          |
| 5   |       |        | error case         |

**Human approval:**
- [ ] Reviewed each example

Approved by: ___  Date: ___

## Step 4: Tests + Implementation (Agent auto-completes)

<!-- Blocked: waiting for Step 2 + Step 3 approval -->

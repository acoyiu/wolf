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

| #   | 輸入 | 輸出 | 說明 |
| --- | ---- | ---- | ---- |
| 1   | `docker build -t wolfword:latest .` | Multi-stage 建置成功。最終映像基於 `alpine`。映像大小 < 30MB。 | 正常：建置 |
| 2   | `docker run -p 3000:3000 wolfword:latest` | 伺服器啟動，log 顯示 `listening on :3000`。`GET /` 回傳 Vue SPA。`ws://localhost:3000/ws` 接受 WebSocket。 | 正常：執行 |
| 3   | `GET /healthz` | 回傳 HTTP 200 `{"status":"ok"}` | 正常：健康檢查 |
| 4   | `docker run -e PORT=8080 wolfword:latest` | 伺服器改用 8080 埠啟動（非預設 3000）。 | 邊界：自訂埠號 |
| 5   | `kubectl apply -f k8s/` | Deployment（replicas=1）+ Service（type=ClusterIP）建立。Pod 運行中且健康。 | 正常：k8s 部署 |
| 6   | Pod 崩潰並重啟 | k8s 重啟 Pod。記憶體中的遊戲狀態丟失（可接受：遊戲時間短）。新連線正常運作。 | 邊界：Pod 重啟 |
| 7   | 主機未安裝 Node.js 時建置 | Multi-stage 建置在 builder 階段處理 Node.js。主機只需要 Docker。 | 邊界：建置依賴 |

**Human approval:**
- [x] Reviewed each example

Approved by: Aco

## Step 4: Tests + Implementation (Agent auto-completes)

<!-- Blocked: waiting for Step 2 + Step 3 approval -->

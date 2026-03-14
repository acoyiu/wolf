# 狼人真言 Web 版（Werewords Online）

## 專案描述

基於桌遊「狼人真言（Werewords）」的網頁版多人即時遊戲。
玩家使用手機瀏覽器訪問 URL 加入遊戲房間，無需安裝 APP。

### 遊戲規則來源
- https://www.drawnow.com.tw/fun/content/102

### 遊戲概要
- 4-10 人派對遊戲，每局約 5 分鐘
- 陣營對抗：村民陣營（村長、先知、村民）vs 狼人陣營（狼人）
- 村長從詞庫選一個「魔法咒語」（秘密詞彙）
- 先知和狼人在夜間偷看咒語
- 白天階段：玩家向村長提問是/否問題，猜測咒語
- 村長只能用指示物回答（是/否/或許/接近了/差太多/正確）
- 猜對咒語 → 狼人有機會指認先知來逆轉勝
- 沒猜對 → 村民投票猜狼人,票數相等的狼人全部開牌，任一為狼人即村民勝

### 技術選型
- **後端**：Go + gorilla/websocket（WebSocket 即時通訊）
- **前端**：Vue 3 + Vite（手機優先響應式設計）
- **部署**：Docker 單一映像，可部署至 Kubernetes
- **資料庫**：無（遊戲狀態存於記憶體）

### 使用方式
- 主持人開啟網頁，建立房間，取得房間代碼/QR 碼
- 其他玩家用手機掃碼或輸入代碼加入
- 所有互動在手機瀏覽器內完成

## Quick Start

### 1. Prerequisites
- Go 1.25+
- Node.js 22+ and npm
- Docker (for container run/deploy)

### 2. Run locally

```bash
# from project root
cd frontend
npm install
npm run build

cd ..
go mod tidy
go run main.go
```

Open: `http://localhost:3000`

Optional env vars:

```bash
PORT=3000 DAY_TIMEOUT_SEC=300 VOTE_TIMEOUT_SEC=60 go run main.go
```

### 3. Run with Docker

```bash
docker build -t wolfword:latest .
docker run --rm -p 3000:3000 wolfword:latest
```

Health check:

```bash
curl http://localhost:3000/healthz
```

### 4. Publish public image to ttl.sh

```bash
chmod +x deploy/push-ttl.sh
./deploy/push-ttl.sh 2h mytag
```

This will:
- build and push image to `ttl.sh`
- patch `k8s/deployment.yaml` image field

Then deploy:

```bash
kubectl apply -f k8s/deployment.yaml -f k8s/service.yaml
```

## 自動化多人測試（免開 4-6 個瀏覽器）

使用內建 WebSocket 機器人煙霧測試，模擬 4 位玩家完整跑一局：

```bash
go test ./internal/ws -run TestHubSmokeFourPlayersNoBrowsers -count=1
```

如果要跑全部後端測試：

```bash
go test ./...
```

## 前端 E2E 測試（免手動開多視窗）

使用 Playwright 在 headless Chromium 中同時模擬 4 位玩家操作 UI：

```bash
cd frontend
npm install
npm run test:e2e:install
npm run test:e2e
```

測試會自動：
- build 前端
- 啟動 Go 後端（本機 `4173`）
- 跑完「建房 -> 4 人加入 -> 開始 -> 夜晚 -> 白天 -> 投票 -> 結果」完整流程

目前前端 E2E 會覆蓋：
- 角色視角驗證：夜晚 step1/step2 不同角色看到的資訊正確
- 投票規則驗證：`guess_seer`（僅狼人投）與 `guess_wolf`（全體可投）
- 刷新恢復驗證：遊戲中重新整理可恢復，且不顯示原始 `resume_*` 錯誤碼
- 結果文案驗證：結果原因顯示為中文說明，不顯示內部 code
- 協作中斷驗證：房主離開房間、玩家斷線重連提示
- 入場失敗驗證：重複暱稱、房間已滿時的中文錯誤提示

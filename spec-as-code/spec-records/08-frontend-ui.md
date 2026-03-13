# Spec: frontend-ui

## Step 1: Context

**Depends on:** room-management, role-assignment, night-phase, day-phase, vote-phase, game-result
**Depended by:** none（前端整合層）
**Related CONTEXT.md:**
- CONTEXT.md (project-level)

**Technical constraints:**
- Vue 3 + Vite + Composition API
- 手機優先（mobile-first）、最小寬度 320px
- 支援 Chrome Mobile 90+、Safari Mobile 14+
- WebSocket 連線狀態管理（連線/斷線/重連）
- 頁面/狀態：首頁（建立/加入房間）→ 等待室 → 夜間 → 白天 → 投票 → 結果
- 觸控友善：大按鈕、無 hover 依賴
- Can build and deploy staticly through golang server

**Scope:**
- Vue 元件結構與路由
- WebSocket 連線管理（composable）
- 各階段畫面 UI
- 響應式佈局
- QR 碼生成（房間分享）

## Step 2: Intent (Human fills in)

Context is totally correct as intent.

## Step 3: Examples (Agent generates → Human approves)

| #   | 輸入 | 輸出 | 說明 |
| --- | ---- | ---- | ---- |
| 1   | 使用者用 375px 寬手機訪問 `/` | 大廳頁面：「建立房間」按鈕 +「加入房間」輸入（房間代碼）+ 暱稱輸入。所有按鈕 ≥44px 觸控區域。 | 正常：手機大廳 |
| 2   | 使用者建立房間 | 等待室頁面：大字顯示房間代碼、QR碼、玩家列表、「開始遊戲」按鈕（僅房主可見）。即時更新加入的玩家。 | 正常：等待室 |
| 3   | 夜間階段開始 | 每位玩家螢幕顯示對應內容（村長選詞介面、先知/狼人看咒語、其他人「下一步」）。全部人都有可點擊按鈕。 | 正常：夜間 UI |
| 4   | 白天階段進行中 | 村長有指示物回應按鈕（✅❌❓❗🚫⭐）、指示物歷史紀錄（可捲動）、倒數計時器、剩餘指示物數量。其他玩家看到指示物歷史與倒數。 | 正常：白天 UI |
| 5   | 投票階段進行中 | 玩家列表可點擊投票、倒數計時器、投票後顯示「已投票」、計時結束後揭曉票數。 | 正常：投票 UI |
| 6   | 遊戲結果顯示 | 勝方公告、所有角色和玩家名稱公開、秘密咒語顯示、關房倒數。 | 正常：結果 UI |
| 7   | WebSocket 意外斷線 | Toast 通知「連線中斷，重新連線中…」。自動重連（最多3次）。失敗則回到大廳並顯示錯誤訊息。 | 邊界：斷線 |
| 8   | 螢幕寬度 = 320px（最小值） | 畫面不溢出、文字可讀、按鈕可點擊。無水平捲軸。 | 邊界：最小寬度 |

**Human approval:**
- [x] Reviewed each example

Approved by: Aco

## Step 4: Tests + Implementation (Agent auto-completes)

<!-- Blocked: waiting for Step 2 + Step 3 approval -->

# Spec: day-phase

## Step 1: Context

**Depends on:** room-management, night-phase
**Depended by:** vote-phase
**Related CONTEXT.md:**
- CONTEXT.md (project-level)

**Technical constraints:**
- 玩家面對面，口頭提問和討論（無需文字聊天）
- 白天計時器：上限 5 分鐘（可配置），倒數同步到所有玩家
- 指示物總數上限：100 個（是 48、否 48、或許 1、接近 1、差太多 1、正確 1）——指示物用完則白天結束
- 村長手機顯示指示物按鈕，點選回應；所有玩家手機上即時顯示指示物歷史與剩餘數
- 玩家口頭猜詞，村長決定是否點「正確」指示物
- 村長點「正確」→ 進入猜先知階段；計時器到或指示物用完 → 進入投票階段

**Scope:**
- 白天問答流程（口頭提問 + 數位指示物回應）
- 村長回應（點選指示物按鈕，包含「正確」）
- 指示物計數與消耗
- 倒數計時器
- 村長點「正確」 / 時間到 / 指示物用完 → 觸發結束條件

## Step 2: Intent (Human fills in)

Context is totally correct as intent.

## Step 3: Examples (Agent generates → Human approves)

| #   | 輸入 | 輸出 | 說明 |
| --- | ---- | ---- | ---- |
| 1   | 白天開始，6人 | 村長手機顯示指示物按鈕：✅是 / ❌否 / ❓或許 / ❗接近 / 🚫差太多 / ⭐正確。所有玩家手機顯示倒數計時器與指示物剩餘數。玩家口頭提問。 | 正常：白天開始 |
| 2   | 村長點擊 ✅(是) | 所有玩家收到 `{"type":"mayor_response","payload":{"token":"yes","remaining":{"yes":47,"no":48,...}}}`。指示物計數遞減。 | 正常：村長回應「是」 |
| 3   | 玩家口頭猜對咒語，村長點擊 ⭐(正確) | 所有人收到 `{"type":"word_guessed","payload":{}}`。進入投票階段（狼人猜先知）。 | 正常：猜對咒語 |
| 4   | 玩家口頭猜錯，村長用一般指示物回應（❌否 或 🚫差太多） | 指示物正常消耗。遊戲繼續。 | 正常：猜錯 |
| 5   | 計時器歸零（5分鐘到） | 所有人收到 `{"type":"time_up","payload":{}}`。進入投票階段（全體猜狼人）。 | 邊界：時間到 |
| 6   | 是/否指示物用完（48+48=96個已用） | 所有人收到 `{"type":"tokens_depleted","payload":{}}`。進入投票階段（全體猜狼人）。 | 邊界：指示物耗盡 |
| 7   | 非村長玩家嘗試發送回應指示物 | 錯誤：`{"type":"error","payload":{"message":"mayor_only"}}`  | 錯誤：無權限 |
| 8   | 白天階段有玩家斷線 | `{"type":"game_aborted","payload":{"reason":"player_disconnected"}}`。遊戲中止。 | 錯誤：斷線 |

**Human approval:**
- [ ] Reviewed each example

Approved by: Aco

## Step 4: Tests + Implementation (Agent auto-completes)

<!-- Blocked: waiting for Step 2 + Step 3 approval -->

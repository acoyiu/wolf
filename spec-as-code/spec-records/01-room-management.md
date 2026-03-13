# Spec: room-management

## Step 1: Context

**Depends on:** none（基礎設施，無依賴）
**Depended by:** role-assignment, night-phase, day-phase, vote-phase
**Related CONTEXT.md:**
- CONTEXT.md (project-level)

**Technical constraints:**
- WebSocket 連線管理（gorilla/websocket）
- 房間狀態存於 Go 記憶體（map）
- 房間代碼：4-6 位英數字，避免混淆字元（0/O, 1/I/L）
- 玩家人數限制：4-10 人
- JSON 訊息格式：`{ "type": string, "payload": object }`

**Scope:**
- 建立房間、加入房間、離開房間
- 房間代碼生成與驗證
- 玩家列表同步（即時更新所有人的畫面）
- 房間主持人（第一個建立者）可開始遊戲
- 房間閒置超時自動清除

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

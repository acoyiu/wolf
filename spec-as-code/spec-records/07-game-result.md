# Spec: game-result

## Step 1: Context

**Depends on:** vote-phase, role-assignment
**Depended by:** none（最終階段）
**Related CONTEXT.md:**
- CONTEXT.md (project-level)

**Technical constraints:**
- 勝利條件判定：
  1. 咒語猜對 + 狼人沒猜中先知 → 村民勝
  2. 咒語猜對 + 狼人猜中先知 → 狼人勝
  3. 咒語沒猜對 + 投票命中狼人 → 村民勝
  4. 咒語沒猜對 + 投票沒命中狼人 → 狼人勝
- 遊戲結束後公開所有玩家角色和咒語
- 遊戲結束強制關閉房間,斷 websocket 連線,需回大廳重新 Gen QR code 開房/加入房

**Scope:**
- 勝負判定邏輯
- 結果展示（勝方、所有角色、咒語）

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

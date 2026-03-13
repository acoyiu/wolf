# Spec: day-phase

## Step 1: Context

**Depends on:** room-management, night-phase
**Depended by:** vote-phase
**Related CONTEXT.md:**
- CONTEXT.md (project-level)

**Technical constraints:**
- 白天計時器：上限 5 分鐘（可配置），倒數同步到所有玩家
- 指示物總數上限：100 個（是 48、否 48、或許 1、接近 1、差太多 1、正確 1）——指示物用完則白天結束
- 村長只能用指示物回應，不能發文字訊息
- 其他玩家可以發文字訊息提問和討論
- 任何玩家可以猜答案（特殊訊息類型）
- 村長回應「正確」→ 進入猜先知階段；計時器到或指示物用完 → 進入投票階段

**Scope:**
- 白天問答流程
- 玩家提問（文字聊天）
- 村長回應（點選指示物按鈕）
- 指示物計數與消耗
- 倒數計時器
- 猜中咒語 / 時間到 / 指示物用完 → 觸發結束條件

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

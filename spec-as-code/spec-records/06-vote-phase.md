# Spec: vote-phase

## Step 1: Context

**Depends on:** room-management, day-phase, role-assignment
**Depended by:** game-result
**Related CONTEXT.md:**
- CONTEXT.md (project-level)

**Technical constraints:**
- 兩種投票場景：
  1. 咒語猜對 → 狼人指認先知（所有狼人投票，1 分鐘）
  2. 咒語沒猜對 → 全體投票猜狼人（1 分鐘）
- 每人只能投一票，不能投自己
- 投票即時同步（可選匿名到時間結束才揭曉，或即時顯示）
- 平票規則：平票玩家全部開牌，任一為狼人即村民勝

**Scope:**
- 投票觸發條件判斷（猜對 vs 沒猜對）
- 投票 UI 與 WebSocket 同步
- 計時器（1 分鐘）
- 票數統計與平票處理
- 投票結果公開

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

# Spec: role-assignment

## Step 1: Context

**Depends on:** room-management（需要房間和玩家列表）
**Depended by:** night-phase
**Related CONTEXT.md:**
- CONTEXT.md (project-level)

**Technical constraints:**
- 角色：Mayor x1, Seer x1, Werewolf x2, Villager x 其餘
- Mayor 由主持人指定（或自動為房間建立者）
- Mayor 額外獲得一個秘密身份（Seer/Werewolf/Villager）
- 隨機分配，每個玩家只能透過 WebSocket 私訊看到自己的角色
- 角色數量依玩家人數調整
- 4-6 人：Mayor + Seer + Werewolf x1 + Villager x (remainings)
- 7-8 人：Mayor + Seer + Werewolf x2 + Villager x (remainings)
- 9-12 人：Mayor + Seer + Werewolf x3 + Villager x (remainings)

**Scope:**
- 根據玩家人數決定角色組成
- 隨機分配角色給每位玩家
- 村長的秘密身份分配
- 透過 WebSocket 私下通知各玩家角色

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

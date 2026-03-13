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

| #   | 輸入                            | 輸出                                                                                                                                                                                              | 說明                        |
| --- | ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------- |
| 1   | 5人遊戲，房主=Alice             | 角色：村長(Alice)+秘密身份、先知x1、狼人x1、村民x2。每人收到私訊 `{"type":"role_assigned","payload":{"role":"villager"}}`。村長額外收到 `{"type":"mayor_secret","payload":{"secretRole":"seer"}}` | 正常：5人（4-6人範圍，1狼） |
| 2   | 8人遊戲，房主=Alice             | 角色：村長(Alice)+秘密身份、先知x1、狼人x2、村民x4。每人私下通知                                                                                                                                  | 正常：8人（7-8人範圍，2狼） |
| 3   | 10人遊戲，房主=Alice            | 角色：村長(Alice)+秘密身份、先知x1、狼人x3、村民x5。每人私下通知                                                                                                                                  | 正常：10人（9+人範圍，3狼） |
| 4   | 4人遊戲（最少人數），房主=Alice | 角色：村長(Alice)+秘密身份、先知x1、狼人x1、村民x1                                                                                                                                                | 邊界：最少人數              |
| 5   | 村長秘密身份隨機抽到狼人        | 村長收到 `{"type":"mayor_secret","payload":{"secretRole":"werewolf"}}`。村長同時是房主和秘密狼人                                                                                                  | 邊界：村長是狼人            |
| 6   | 村長秘密身份隨機抽到先知        | 村長收到 `{"type":"mayor_secret","payload":{"secretRole":"seer"}}`。村長同時是房主和秘密先知                                                                                                      | 邊界：村長是先知            |
| 7   | 只有3人時觸發開始遊戲           | 錯誤：由 room-management 攔截（人數不足），role-assignment 不會被呼叫                                                                                                                             | 錯誤：前置條件不符          |

**Human approval:**
- [x] Reviewed each example

Approved by: Aco

## Step 4: Tests + Implementation (Agent auto-completes)

<!-- Blocked: waiting for Step 2 + Step 3 approval -->

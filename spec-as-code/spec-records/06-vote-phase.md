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

| #   | 輸入                                                         | 輸出                                                                                                                                                                                   | 說明               |
| --- | ------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------ |
| 1   | 咒語猜對 → 狼人投票猜先知。狼人1投給Carol                    | `{"type":"vote_cast","payload":{"voter":"Wolf1","target":"Carol"}}` 發送到伺服器。其他玩家看到票數更新（計時結束前匿名）。                                                             | 正常：狼人投票     |
| 2   | 投票計時（1分鐘）結束。狼人1和狼人2都投Carol，Carol是先知    | `{"type":"vote_result","payload":{"topVoted":["Carol"],"revealedRole":"seer","isCorrect":true}}`。狼人勝（猜中先知）。                                                                 | 正常：狼人找到先知 |
| 3   | 咒語沒猜對 → 全體投票猜狼人。Bob(3票)、Carol(2票)。Bob是狼人 | `{"type":"vote_result","payload":{"topVoted":["Bob"],"revealedRole":"werewolf","isCorrect":true}}`。村民勝。                                                                           | 正常：村民抓到狼人 |
| 4   | 咒語沒猜對 → 全體投票。Bob(3票)、Carol(2票)。Bob是村民       | `{"type":"vote_result","payload":{"topVoted":["Bob"],"revealedRole":"villager","isCorrect":false}}`。狼人勝。                                                                          | 正常：狼人存活     |
| 5   | 平票：Bob(3票)、Carol(3票)。Bob=村民、Carol=狼人             | 兩人都開牌：`{"type":"vote_result","payload":{"topVoted":["Bob","Carol"],"revealedRoles":[{"Bob":"villager"},{"Carol":"werewolf"}],"isCorrect":true}}`。村民勝（平票中有狼人即抓到）。 | 邊界：平票有狼人   |
| 6   | 平票：Bob(3票)、Carol(3票)。兩人都是村民                     | 兩人都開牌。沒有狼人。`isCorrect: false`。狼人勝。                                                                                                                                     | 邊界：平票無狼人   |
| 7   | 玩家嘗試投自己                                               | 錯誤：`{"type":"error","payload":{"message":"cannot_vote_self"}}`                                                                                                                      | 錯誤：投自己       |
| 8   | 玩家嘗試重複投票                                             | 錯誤：`{"type":"error","payload":{"message":"already_voted"}}`                                                                                                                         | 錯誤：重複投票     |
| 9   | 計時結束，部分玩家未投票                                     | 伺服器為未投票者隨機選擇一個目標（不能是自己）。所有票數一起結算。                                                                                                                     | 邊界：超時未投票   |

**Human approval:**
- [ ] Reviewed each example

Approved by: Aco

## Step 4: Tests + Implementation (Agent auto-completes)

<!-- Blocked: waiting for Step 2 + Step 3 approval -->

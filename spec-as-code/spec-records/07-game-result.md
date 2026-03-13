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

| #   | 輸入                         | 輸出                                                                                                                                         | 說明                     |
| --- | ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------ |
| 1   | 咒語猜對 + 狼人沒猜中先知    | `{"type":"game_over","payload":{"winner":"villagers","reason":"word_guessed_seer_safe","word":"apple","roles":{...}}}`。公開所有角色和咒語。 | 正常：村民勝（猜對咒語） |
| 2   | 咒語猜對 + 狼人猜中先知      | `{"type":"game_over","payload":{"winner":"werewolves","reason":"word_guessed_seer_found","word":"apple","roles":{...}}}`                     | 正常：狼人逆轉勝         |
| 3   | 咒語沒猜對 + 投票命中狼人    | `{"type":"game_over","payload":{"winner":"villagers","reason":"word_missed_wolf_caught","word":"apple","roles":{...}}}`                      | 正常：村民勝（投票）     |
| 4   | 咒語沒猜對 + 投票沒命中狼人  | `{"type":"game_over","payload":{"winner":"werewolves","reason":"word_missed_wolf_safe","word":"apple","roles":{...}}}`                       | 正常：狼人勝             |
| 5   | 遊戲結果顯示10秒後           | 所有 WebSocket 關閉。房間刪除。所有玩家回到大廳。`{"type":"room_closed","payload":{"reason":"game_ended"}}`                                  | 邊界：強制關房           |
| 6   | 玩家在結果畫面重新整理瀏覽器 | 玩家看到大廳頁面。房間已不存在。                                                                                                             | 邊界：結束後重新整理     |

**Human approval:**
- [x] Reviewed each example

Approved by: Aco

## Step 4: Tests + Implementation (Agent auto-completes)

<!-- Blocked: waiting for Step 2 + Step 3 approval -->

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
- 房主畫面顯示 QR Code（包含加入房間的 URL），人數到齊後顯示「開始遊戲」按鈕
- 玩家列表同步（即時更新所有人的畫面）
- 房間主持人（第一個建立者）可開始遊戲
- 房間閒置超時自動清除

## Step 2: Intent (Human fills in)

Context is totally correct as intent.

## Step 3: Examples (Agent generates → Human approves)

| #   | 輸入                                                                                 | 輸出                                                                                                                                                            | 說明                     |
| --- | ------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------ |
| 1   | 玩家A 發送 `{"type":"create_room","payload":{"nickname":"Alice","targetPlayers":6}}` | 伺服器回傳 `{"type":"room_created","payload":{"roomCode":"AB3K","targetPlayers":6,"players":[{"id":"...","nickname":"Alice","isHost":true}]}}`。房主畫面顯示 QR Code（包含加入 URL）+ 玩家列表。 | 正常：建立房間           |
| 2   | 玩家B 發送 `{"type":"join_room","payload":{"roomCode":"AB3K","nickname":"Bob"}}`     | 所有玩家收到 `{"type":"player_joined","payload":{"players":[Alice, Bob]}}`。若人數到達 targetPlayers，房主畫面顯示「開始遊戲」按鈕。                                                                                      | 正常：加入房間           |
| 3   | 玩家B 發送 `{"type":"leave_room","payload":{}}`                                      | 剩餘玩家收到 `{"type":"player_left","payload":{"players":[Alice]}}`                                                                                             | 正常：離開房間           |
| 4   | 玩家C 用已存在的暱稱 "Alice" 加入房間                                                | 錯誤：`{"type":"error","payload":{"message":"nickname_already_taken"}}`                                                                                         | 邊界：暱稱重複           |
| 5   | 第7人加入 targetPlayers=6 且已有6人的房間                                            | 錯誤：`{"type":"error","payload":{"message":"room_full"}}`                                                                                                      | 邊界：房間已滿           |
| 6   | 玩家用不存在的房間代碼 "ZZZZ" 加入                                                   | 錯誤：`{"type":"error","payload":{"message":"room_not_found"}}`                                                                                                 | 錯誤：無效房間代碼       |
| 7   | 房主在只有2人時開始遊戲（targetPlayers=6）                                           | 錯誤：`{"type":"error","payload":{"message":"not_enough_players"}}`                                                                                              | 錯誤：人數不足           |
| 8   | 非房主玩家嘗試開始遊戲                                                               | 錯誤：`{"type":"error","payload":{"message":"host_only"}}`                                                                                                      | 錯誤：非房主操作         |
| 9   | 房間閒置超時（例如5分鐘無活動）                                                      | 房間自動刪除，所有 WebSocket 關閉，收到 `{"type":"room_closed","payload":{"reason":"idle_timeout"}}`                                                             | 邊界：自動清理           |
| 10  | 房主斷線（WebSocket 關閉）                                                           | 房間刪除，所有連線關閉，所有玩家收到 `{"type":"room_closed","payload":{"reason":"host_disconnected"}}` 並回到大廳                                                | 邊界：房主斷線，遊戲作廢 |

**Human approval:**
- [x] Reviewed each example

Approved by: Aco

## Step 4: Tests + Implementation (Agent auto-completes)

<!-- Blocked: waiting for Step 2 + Step 3 approval -->

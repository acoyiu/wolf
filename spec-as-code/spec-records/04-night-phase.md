# Spec: night-phase

## Step 1: Context

**Depends on:** room-management, role-assignment, word-library
**Depended by:** day-phase
**Related CONTEXT.md:**
- CONTEXT.md (project-level)

**Technical constraints:**
- 防偷看機制：每一步**所有玩家都必須點擊螢幕**，防止旁人觀察誰在操作來推測角色
- 每一步伺服器發送不同內容給不同角色，但 UI 外觀一致（都是「點擊確認」按鈕）
  - 特定角色：看到真實內容（如村長看到選詞介面、先知看到咒語）
  - 其他玩家：看到掩護內容（如「請點擊確認以繼續」），點擊後等待
- 伺服器等待**全員**都點擊後才進入下一步（不會因為某人先點完就暴露身份）
- 流程：村長選咒語（其他人看「下一步」）→ 全員確認 → 先知和狼人同時看咒語（其他人看「下一步」）→ 全員確認 → 進入白天

**Scope:**
- 夜間流程狀態機（每步等待全員點擊）
- 村長選擇咒語（從詞庫中呈現候選詞，村長點選；其他人只看到「下一步」按鈕）
- 先知和狼人同時收到咒語（其他人看到「下一步」）
- 每步全員點擊後才推進下一步
- 全員收到「白天開始」→ 轉入白天階段

## Step 2: Intent (Human fills in)

Context is totally correct as intent.

## Step 3: Examples (Agent generates → Human approves)

| #   | 輸入 | 輸出 | 說明 |
| --- | ---- | ---- | ---- |
| 1   | 遊戲開始，6人，難度=簡單 | 第1步：村長看到3個候選詞 `["蘋果","太陽","書包"]`。其他人看到「下一步」。伺服器等待全部6人點擊。 | 正常：村長選詞 |
| 2   | 村長選擇「蘋果」 | 第2步：先知和狼人同時看到 `{"type":"night_reveal","payload":{"word":"蘋果"}}` + 確認按鈕。其他人看到「下一步」。伺服器等待全部6人點擊。 | 正常：先知+狼人看到咒語 |
| 3   | 所有人確認 | 所有玩家收到 `{"type":"phase_change","payload":{"phase":"day"}}`。夜間階段結束。 | 正常：進入白天 |
| 4   | 村長的秘密身份是先知 | 村長在第1步選詞，已知詞。第2步（先知+狼人看詞），村長看到「下一步」（無需重複顯示）。伺服器仍等待全員點擊。 | 邊界：村長是先知 |
| 5   | 村長的秘密身份是狼人 | 村長在第1步選詞，已知詞。第2步（先知+狼人看詞），村長看到「下一步」（無需重複顯示）。 | 邊界：村長是狼人 |
| 6   | 村長的秘密身份是村民 | 村長在第1步選詞，已知詞。第2步（先知+狼人看詞），村長看到「下一步」（無需重複顯示）。行為與其他情況一致。 | 邊界：村長是村民 |
| 7   | 夜間有玩家斷線 | 伺服器發送 `{"type":"game_aborted","payload":{"reason":"player_disconnected"}}` 給所有人。遊戲中止，回到大廳。 | 錯誤：斷線 |

**Human approval:**
- [ ] Reviewed each example

Approved by: Aco

## Step 4: Tests + Implementation (Agent auto-completes)

<!-- Blocked: waiting for Step 2 + Step 3 approval -->

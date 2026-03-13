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
- 流程：全員點擊確認 → 村長選咒語（其他人假裝選） → 全員確認 → 先知看咒語（其他人看掩護） → 全員確認 → 狼人看咒語（其他人看掩護） → 全員確認 → 進入白天

**Scope:**
- 夜間流程狀態機（每步等待全員點擊）
- 村長選擇咒語（從詞庫中呈現候選詞，村長點選；其他人只看到「下一步」按鈕）
- 先知收到咒語（其他人看到掩護文字）
- 狼人收到咒語（其他人看到掩護文字）
- 每步全員點擊後才推進下一步
- 全員收到「白天開始」→ 轉入白天階段

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

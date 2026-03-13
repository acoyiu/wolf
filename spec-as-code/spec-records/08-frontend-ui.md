# Spec: frontend-ui

## Step 1: Context

**Depends on:** room-management, role-assignment, night-phase, day-phase, vote-phase, game-result
**Depended by:** none（前端整合層）
**Related CONTEXT.md:**
- CONTEXT.md (project-level)

**Technical constraints:**
- Vue 3 + Vite + Composition API
- 手機優先（mobile-first）、最小寬度 320px
- 支援 Chrome Mobile 90+、Safari Mobile 14+
- WebSocket 連線狀態管理（連線/斷線/重連）
- 頁面/狀態：首頁（建立/加入房間）→ 等待室 → 夜間 → 白天 → 投票 → 結果
- 觸控友善：大按鈕、無 hover 依賴
- Can build and deploy staticly through golang server

**Scope:**
- Vue 元件結構與路由
- WebSocket 連線管理（composable）
- 各階段畫面 UI
- 響應式佈局
- QR 碼生成（房間分享）

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

# Spec: word-library

## Step 1: Context

**Depends on:** none（獨立資料模組）
**Depended by:** night-phase（村長從詞庫選詞）
**Related CONTEXT.md:**
- CONTEXT.md (project-level)

**Technical constraints:**
- 詞庫為靜態資料，嵌入 Go 二進位檔案（embed）或 JSON 檔案
- 難度分級：簡單 / 中等 / 困難
- 語言：中文（預設），未來可擴展
- 每個難度至少 100 個詞彙
- 每次選擇可在 3 個候選詞中選 1 個

**Scope:**
- 詞庫資料結構定義
- 依難度隨機取詞候選詞
- 詞庫載入

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

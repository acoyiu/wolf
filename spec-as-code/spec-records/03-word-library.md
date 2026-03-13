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

| #   | 輸入                                                 | 輸出                                                         | 說明             |
| --- | ---------------------------------------------------- | ------------------------------------------------------------ | ---------------- |
| 1   | `GetCandidates(difficulty="easy")`                   | 回傳3個隨機簡單詞，例如 `["蘋果", "太陽", "書包"]`           | 正常：簡單難度   |
| 2   | `GetCandidates(difficulty="medium")`                 | 回傳3個隨機中等詞，例如 `["民主", "引力", "咖啡因"]`         | 正常：中等難度   |
| 3   | `GetCandidates(difficulty="hard")`                   | 回傳3個隨機困難詞，例如 `["量子纏結", "存在主義", "區塊鏈"]` | 正常：困難難度   |
| 4   | 同一局內呼叫 `GetCandidates(difficulty="easy")` 兩次 | 每次回傳不同的3個詞（同局內不重複）                          | 邊界：不重複     |
| 5   | `GetCandidates(difficulty="invalid")`                | 錯誤：`invalid difficulty level`                             | 錯誤：無效難度   |
| 6   | 程式啟動時載入詞庫                                   | 每個難度至少100個詞；驗證詞彙總數                            | 邊界：詞庫完整性 |

**Human approval:**
- [x] Reviewed each example

Approved by: Aco

## Step 4: Tests + Implementation (Agent auto-completes)

<!-- Blocked: waiting for Step 2 + Step 3 approval -->

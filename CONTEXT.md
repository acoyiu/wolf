# CONTEXT.md — 狼人真言 Web 版

## 專案級規則

### 技術約束
- 後端語言：Go 1.21+
- 前端框架：Vue 3 + Vite
- 通訊協定：WebSocket（gorilla/websocket/Gin[http]）
- 部署：單一 Docker 映像（multi-stage build），Go 伺服器同時服務 API 和前端靜態檔案
- 無資料庫：所有遊戲狀態存於 Go 記憶體（房間結束即清除）
- 無持久化需求：不儲存歷史遊戲記錄

### 遊戲約束
- 玩家人數：4-10 人
- 角色：村長(Mayor) x1、先知(Seer) x1、狼人(Werewolf) x2、村民(Villager) x 其餘
- 村長同時擁有秘密身份（可能是先知/狼人/村民）
- 指示物總數上限：100 個（是/否各 48，[或許/接近了/差太多/正確] 各 1）
- 每局時間限制：白天問答階段上限 5 分鐘（可配置）
- 投票階段：1 分鐘
- 村民投票猜狼人,票數相等的狼人全部開牌，任一為狼人即村民勝
- 咒語詞庫：預設中文詞庫，支援難度分級（簡單/中等/困難）

### 前端約束
- 手機優先（mobile-first）響應式設計
- 最小支援寬度：320px
- 支援瀏覽器：Chrome Mobile 90+、Safari Mobile 14+
- 不使用原生 APP 功能（如推播），純網頁

### 編碼慣例
- Go：遵循 standard Go project layout，使用 `internal/` 放內部套件
- Vue：使用 Composition API（setup script），不要使用 TypeScript,但要有嚴謹的 JSDoc 註解
- WebSocket 訊息格式：JSON，結構為 `{ "type": string, "payload": object }`
- 測試：Go 用 `go test`，前端用 `vitest`

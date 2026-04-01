---
name: go-style-guide
description: 遵循 Uber 與 Google Go 風格指南編寫地道 (idiomatic) 且具可讀性的 Go 程式碼。本指南以 Google Style 為核心原則，並結合 Uber Style 的實踐建議。
---

# Go 語言風格指南 (Integrated Style Guide)

本技能旨在引導開發者編寫清晰、簡潔、具可維護性且符合 Go 地道用法 (idiomatic) 的程式碼。

## 核心指導原則 (Core Principles)

應優先考慮以下屬性（按重要性排序）：

1. **清晰 (Clarity)**：程式碼的意圖和原理對讀者來說是明確的。優先考慮讀者而非作者。
2. **簡單 (Simplicity)**：以最簡單的方式實現目標，避免過度抽象或「聰明」的代碼。
3. **簡練 (Concision)**：具備高信噪比，避免冗餘註釋或重複代碼。
4. **可維護性 (Maintainability)**：易於長期維護，包含良好的測試與有意義的錯誤處理。
5. **一致性 (Consistency)**：與整體程式碼庫風格保持一致。

> **優先級聲明**：本指南參考了 Google 與 Uber 的規範。若兩者規則衝突，請以 **Google Style** 為準。

## 1. 命名規範 (Naming)

- **MixedCaps**：所有標識符均使用 PascalCase (導出) 或 camelCase (非導出)。禁用底線（測試、cgo 或特殊生成代碼除外）。
- **包名 (Package Names)**：
    - 僅使用小寫字母和數字，不使用底線。
    - 應簡潔且具描述性，避免 `util`, `common`, `helper` 等無意義名稱。
- **變數名 (Variable Names)**：
    - 傾向短變數名，尤其是範圍 (scope) 較小時。
    - 接收者 (Receiver) 名稱應極短（1-2 個字母），且為類型的縮寫。
- **常量 (Constants)**：
    - 使用 MixedCaps，不使用全大寫加底線 (SCREAMING_SNAKE_CASE)。
    - 不使用 `K` 前綴。

## 2. 編程實踐 (Programming Practices)

- **接口合規性檢查 (Interface Compliance)**：
    - 在編譯時驗證類型是否實現了接口：`var _ http.Handler = (*Handler)(nil)`。
- **減少嵌套 (Reduce Nesting)**：
    - 優先處理錯誤/特殊情況並儘早返回 (Early Return)，以保持快樂路徑 (Happy Path) 在左側。
- **切片與映射 (Slices and Maps)**：
    - 在邊界處（如函數輸入/輸出）拷貝切片或映射，避免外部修改影響內部狀態。
    - 預先指定容器容量 (Capacity) 以提高效能。
- **結構體初始化**：
    - 初始化時應明確指定欄位名稱：`User{ID: 1, Name: "Alice"}`。
    - 零值結構體優先使用 `var` 聲明。

## 3. 錯誤處理 (Error Handling)

- **處理一次**：錯誤應被處理（記錄、重試或返回）且僅處理一次。
- **錯誤包裝 (Error Wrapping)**：
    - 使用 `%w` 進行錯誤包裝以保留原始上下文。
    - 導出錯誤應以 `Err` 為前綴，例如 `ErrNotFound`。
- **不要 Panic**：除非在 `main` 函數或不可恢復的初始化中，否則應返回 `error`。

## 4. 並發與效能 (Concurrency & Performance)

- **Channel 緩衝**：Channel 大小通常應為 0 或 1。
- **Goroutine 生命週期**：不要「發射後不管」(fire-and-forget)，必須確保 Goroutine 能夠被正確關閉或等待其完成。
- **避免可變全局變量**：避免使用全局狀態，優先使用依賴注入。
- **效能優化**：
    - 優先使用 `strconv` 而非 `fmt` 進行類型轉換。
    - 避免重複的 string-to-byte 轉換。

## 5. 模式與工具 (Patterns & Tools)

- **Functional Options**：對於具有多個可選配置的構造函數，優先使用 Functional Options 模式。
- **表格驅動測試 (Table-driven Tests)**：使用子測試 (`t.Run`) 編寫簡潔且覆蓋面廣的測試案例。
- **工具鏈**：
    - 必須通過 `gofmt` 格式化。
    - 使用 `go vet` 和 `staticcheck` (或 `golangci-lint`) 檢查潛在問題。

## 詳細參考資源

- **Google Style**: [Guide](references/google/guide.md), [Decisions](references/google/decisions.md), [Best Practices](references/google/best-practices.md)
- **Uber Style**: [Style Guide](references/uber/style.md)

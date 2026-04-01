---
name: go-project-layout
description: 遵循社群公認的標準 Go 專案佈局 (Standard Go Project Layout) 組織代碼，確保專案具備良好的可擴展性、封裝性與清晰度。
---

# Go 專案佈局規範 (Standard Go Project Layout)

本技能旨在引導開發者按照 Go 生態系統中廣泛採用的模式來組織專案結構。這不是官方標準，但是維護大型且實用 Go 應用程式的最佳實踐。

## 核心指導原則 (Core Principles)

1. **意圖清晰 (Express Intent)**：透過目錄結構明確表達程式碼的用途（例如：執行檔、私有庫、公共庫）。
2. **封裝與保護 (Encapsulation)**：利用 `internal` 目錄強制執行存取控制，防止非預期的相依性。
3. **簡單起步 (Start Simple)**：小型專案應從簡單結構開始，僅在專案增長時才引入複雜的目錄層級。
4. **地道用法 (Idiomatic)**：避免使用其他語言（如 Java 的 `src`）的慣用法，遵循 Go 的命名與組織習慣。

## 1. 核心代碼目錄 (Core Code Directories)

- **`/cmd`**：
    - 存放專案的主要執行檔。
    - 每個應用程式一個子目錄，名稱應與執行檔一致（如 `/cmd/myapp`）。
    - 此目錄應僅包含極少量的 `main` 函數代碼，主要邏輯應從 `internal` 或 `pkg` 導入。
- **`/internal`**：
    - 存放**私有**應用程式與函式庫代碼。
    - Go 編譯器會強制執行限制，此目錄下的代碼無法被外部專案導入。
    - 適用於不希望公開的商業邏輯或內部組件。
- **`/pkg`**：
    - 存放**可導出**的函式庫代碼。
    - 外部專案可以安全地導入並使用此目錄下的代碼。
    - **注意**：如果專案很小，或者代碼不打算給別人用，優先使用 `internal`。

## 2. 服務與 Web 目錄 (Service & Web)

- **`/api`**：OpenAPI/Swagger 規格、JSON Schema、Proto 定義檔。
- **`/web`**：靜態檔案、伺服器端模板、SPA 相關組件。

## 3. 支撐與配置目錄 (Support & Configs)

- **`/configs`**：配置設定範本或預設值。
- **`/scripts`**：各類建置、分析、安裝用的腳本。
- **`/build`**：打包與持續整合 (CI) 相關。`/build/package` 存放 Docker/AMI 等打包配置；`/build/ci` 存放 CI 腳本。
- **`/deployments`**：IaaS/PaaS 部署配置（如 Kubernetes, Terraform）。

## 4. 測試與開發工具 (Testing & Tools)

- **`/test`**：額外的外部測試應用程式與測試數據（如 `/test/data`）。
- **`/tools`**：專案的支援工具，可從 `internal` 或 `pkg` 導入代碼。
- **`/examples`**：應用程式或函式庫的使用範例。

## 5. 其他資源 (Miscellaneous)

- **`/assets`**：圖片、Logo 等靜態資源。
- **`/docs`**：設計文件、用戶指南（非自動產生的 Godoc）。
- **`/vendor`**：相依套件副本（在使用 Go Modules 時通常不需要，除非有特殊離線建置需求）。

## 禁忌事項 (Anti-Patterns)

- **避免 `/src`**：專案根目錄不應包含 `src` 目錄，這是不符合 Go 慣例的 Java 式結構。
- **避免過度工程**：對於微型專案或實驗性工具，單個 `main.go` 或簡單結構即可。

## 詳細參考資源

- **完整指南 (繁中)**：[Standard Layout (zh-TW)](references/project-layout/README_zh-TW.md)
- **完整指南 (簡中)**：[Standard Layout (zh-CN)](references/project-layout/README_zh-CN.md)
- **英文原版**：[Standard Layout (English)](references/project-layout/README.md)

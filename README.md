# 证据卫士 (Evidence Guardian)

劳动者权益保护取证系统 — 自动收集工作过程中的关键证据。

## 功能

- **关键词触发**：监控屏幕内容，发现预设关键词自动取证
- **多应用支持**：Chrome、Edge、Firefox、企业微信、钉钉、QQ、微信、飞书
- **截图取证**：捕获目标应用窗口的实时画面（所见即所得）
- **视频录制**：5-10秒窗口录制，保留动态操作痕迹
- **DPAPI加密**：证据文件自动加密，仅本机本用户可解密
- **用户自定义**：关键词、目标应用、采集策略均可配置

## 架构

```
evidence-guardian/
├── cmd/service/       Windows 服务入口
├── internal/
│   ├── trigger/       触发引擎（标题/OCR/热键）
│   ├── capture/       采集模块（截图/视频/窗口）
│   ├── ocr/           OCR 文字识别引擎
│   ├── storage/       存储管理（索引/配额）
│   ├── crypto/        DPAPI 加密
│   └── config/        配置管理
└── ui/                系统托盘/配置面板/证据浏览器
```

## 构建

```powershell
go build -o bin/evidence-guardian.exe .
```

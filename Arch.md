# 证据卫士 — 架构决策记录

## 项目定位

劳动者权益保护取证系统。自动监测屏幕上的敏感关键词（调薪、转岗、辞退等），触发截图和视频录制，为劳动者留存证据。

## 核心架构

### 三级文本检测（触发层）

```
L1 标题检测 (2s) ── GetWindowText 取所有窗口标题 → 关键词匹配
    100%准确，零开销，覆盖浏览器标签/IM对话标题/文档名

L2 浏览器CDP (可选) ── Chrome DevTools Protocol → document.body.innerText
    100%准确，仅限 Chrome/Edge，需浏览器启动时加 --remote-debugging-port

L3 OCR兜底 (10-30s) ── 截取窗口 → Tesseract OCR → 关键词匹配
    ~90-95%准确，覆盖IM聊天内容等无法通过API获取文本的场景
```

### 三级采集模式（采集层）

| 模式 | 说明 | 默认 |
|------|------|------|
| `browser` | 仅监控浏览器窗口标题 + CDP 页面内容 | ✅ |
| `desktop` | 浏览器 + 全桌面 OCR 巡检 | |
| `full` | 桌面 OCR + 逐窗口截取 | |

### 三级用户通知

| 模式 | 行为 |
|------|------|
| `silent` | 静默采集，无任何提示 |
| `toast` | Windows 系统通知气泡 |
| `alert` | 弹窗提醒用户注意 |

## 交互方式

- **后台服务**：系统托盘常驻（蓝色盾牌图标）
- **管理面板**：本地 Web 页面 `http://127.0.0.1:58080`
- **手动触发**：托盘菜单 / 全局热键 `Ctrl+Shift+F12`
- **开机自启**：注册表 `HKCU\...\Run`

## 目标应用识别表

| 应用 | 进程名 | 窗口类名 |
|------|--------|----------|
| Chrome | `chrome.exe` | `Chrome_WidgetWin_1` |
| Edge | `msedge.exe` | `Chrome_WidgetWin_1` |
| Firefox | `firefox.exe` | `MozillaWindowClass` |
| 企业微信 | `WXWork.exe` | `WXWorkMainWindow` |
| 钉钉 | `DingTalk.exe` | `DingTalk` |
| QQ | `QQ.exe` | `TXGuiFoundation` |
| 微信 | `WeChat.exe` | `WeChatMainWndForPC` |
| 飞书 | `Feishu.exe` | `Chrome_WidgetWin_0` |
| 自定义 | 用户配置 | 用户配置 |

## 关键决策

### 为什么用 OCR 而不是 API 取 IM 聊天内容？

微信/企微/钉钉/QQ 全部使用自定义渲染引擎（DirectUI/Skia/CEF），不走 Windows 标准文本控件。UI Automation、MSAA 等无障碍 API 均无法提取聊天消息内容。OCR 是唯一可行的方案。

### 为什么用 Tesseract 而不是 Windows.Media.Ocr？

Windows.Media.Ocr 需要 MSIX 打包才能使用，对传统 Go 二进制不友好。Tesseract 通过 `gosseract` Go 绑定可直接使用，无需额外运行时依赖。

### 为什么 DXGI 而不是 PrintWindow 截图？

PrintWindow 在 Chromium 和 IM 应用上常返回黑屏/空白。Desktop Duplication API（DXGI）直接从显卡抓取合成后的桌面，所见即所得，能捕获弹窗、右键菜单、硬件加速内容。

### 为什么用 DPAPI 加密？

Windows 内置 DPAPI（CryptProtectData）无需管理密码，密钥从当前用户登录凭据自动派生。加密后的文件仅本机本用户可解密。零外部依赖。

### IM 应用为什么只能捕获窗口标题而非聊天内容？

这是当前方案的核心局限。以下为各 IM 的技术分析：

| 应用 | UI框架 | 聊天内容API可访问性 |
|------|--------|-------------------|
| 微信 | 自研 DirectUI | ❌ 不可访问 |
| 企业微信 | 自研 DirectUI | ❌ 不可访问 |
| 钉钉 | Electron + 自定义 | ⚠️ 部分版本不可访问 |
| QQ | 自研渲染引擎 | ❌ 不可访问 |

OCR 是唯一能在内容层面检测关键词的手段，但准确率约 90-95%，且受窗口重叠影响。

## 存储结构

```
evidence/
├── index.db              SQLite 索引（明文，仅存元数据路径和时间戳）
├── 2025-05-25/
│   ├── 093000_调薪/
│   │   ├── chrome.png.enc     DPAPI 加密截图
│   │   ├── wework.mp4.enc     DPAPI 加密视频
│   │   └── metadata.json.enc  元数据
```

## Go 技术栈

| 能力 | 方案 | CGO |
|------|------|-----|
| Win32 API | `golang.org/x/sys/windows` | ❌ |
| 系统托盘 | `github.com/getlantern/systray` | ❌ |
| 配置解析 | `gopkg.in/yaml.v3` | ❌ |
| DPAPI加密 | `crypt32.dll` 直接 syscall | ❌ |
| 全局热键 | `user32.dll` RegisterHotKey | ❌ |
| 窗口枚举 | `user32.dll` EnumWindows | ❌ |
| 截图 | Desktop Duplication API (DXGI) | ❌ |
| OCR | Tesseract (gosseract) | ✅ |
| 浏览器文本 | chromedp (CDP) | ❌ |
| 视频录制 | FFmpeg 子进程管道 | ❌ |
| 证据索引 | SQLite (mattn/go-sqlite3) | ✅ |
| Web管理页 | Go net/http + embed | ❌ |

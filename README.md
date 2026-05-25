# 证据卫士 (Evidence Guardian)

劳动者权益保护取证系统。自动监测工作屏幕上的敏感关键词，触发截图和视频录制，留存关键证据。

## 功能

- **关键词触发** — 监测窗口标题和屏幕内容，匹配预设关键词自动取证
- **三级采集模式** — 仅浏览器 / 桌面+浏览器 / 全面采集，按需配置
- **多应用支持** — Chrome、Edge、Firefox、企业微信、钉钉、QQ、微信、飞书
- **所见即所得截图** — Desktop Duplication API 直接捕获用户看到的画面
- **视频录制** — 5-10 秒窗口录制，保留动态操作痕迹
- **DPAPI 加密** — 证据文件自动加密，仅本机本用户可解密
- **Web 管理面板** — 浏览器访问 `http://127.0.0.1:58080` 配置和管理
- **全局热键** — `Ctrl+Shift+F12` 一键手动取证
- **通知模式** — 静默采集 / 系统通知 / 弹窗提醒

## 快速开始

### 编译

```powershell
cd evidence-guardian
$env:GOPROXY="https://goproxy.cn,direct"
go build -o evidence-guardian.exe .
```

### 运行

```powershell
.\evidence-guardian.exe
```

首次运行自动生成 `config.yaml`。程序在系统托盘显示盾牌图标。

### 管理面板

运行后在浏览器打开 `http://127.0.0.1:58080`，可配置关键词、采集模式、查看状态。

### 手动取证

- 托盘菜单 → 立即取证
- 快捷键 `Ctrl+Shift+F12`

## 配置

编辑 `config.yaml` 或通过管理面板修改：

```yaml
keywords:
  - 调薪
  - 降薪
  - 转岗
  - 辞退

capture_mode: browser    # browser | desktop | full
notify_on_trigger: silent # silent | toast | alert

ocr:
  interval_sec: 10       # OCR 检测间隔（秒）
  enabled: true

capture:
  video_duration_sec: 8  # 视频录制时长
  video_fps: 10

storage:
  path: ./evidence
  max_size_gb: 50
  encrypt: true
```

## 架构概览

```
┌─ 触发层 ─────────────────────────────┐
│  L1 标题检测 (2s)                      │
│  L2 CDP 浏览器内容 (Chrome/Edge)       │
│  L3 OCR 屏幕内容 (Tesseract)           │
└──────────┬─────────────────────────────┘
           ▼ 任一关键词命中
┌─ 采集层 ─────────────────────────────┐
│  截图 (DXGI Desktop Duplication)       │
│  视频 (FFmpeg)                         │
│  元数据记录                             │
└──────────┬─────────────────────────────┘
           ▼
┌─ 存储层 ─────────────────────────────┐
│  DPAPI 加密文件                        │
│  SQLite 证据索引                       │
│  磁盘配额管理                           │
└────────────────────────────────────────┘
```

详细架构决策见 [Arch.md](Arch.md)。

## 技术栈

| 语言 | Go 1.21+ |
| 平台 | Windows 10 / 11 |
| 截图 | Desktop Duplication API (DXGI) |
| OCR | Tesseract (gosseract) |
| 视频 | FFmpeg 管道 |
| 加密 | Windows DPAPI |
| 存储 | SQLite + 文件系统 |
| 界面 | 系统托盘 + Web 管理面板 |

## 项目结构

```
evidence-guardian/
├── main.go                  入口
├── internal/
│   ├── config/              配置定义和加载
│   ├── trigger/             触发引擎（标题/热键/OCR轮询）
│   ├── capture/             采集模块（截图/视频/窗口）
│   ├── ocr/                 OCR 引擎接口
│   ├── crypto/              DPAPI 加解密
│   ├── storage/             存储管理和索引
│   └── icon/                托盘图标生成
├── ui/
│   ├── tray/                系统托盘
│   ├── server/              Web 管理服务器
│   ├── panel/               配置面板（待实现）
│   └── viewer/              证据浏览器（待实现）
├── Arch.md                  架构决策记录
├── AI.md                    AI 开发约定
└── README.md
```

## 已知局限

- IM 聊天消息内容只能通过 OCR 识别，无法直接获取文本
- OCR 受窗口重叠影响（被挡的文字无法识别）
- 视频录制依赖系统安装 FFmpeg

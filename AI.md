# AI 开发约定

本文档供 AI 编码助手参考，确保生成代码符合项目规范。

## 目录规范

```
evidence-guardian/
├── main.go                  入口，初始化各模块
├── internal/                内部包，不对外暴露
│   ├── config/              配置结构体定义 + YAML 加载
│   ├── trigger/             触发引擎（标题/热键/OCR）
│   ├── capture/             采集模块（截图/视频/窗口枚举）
│   ├── ocr/                 OCR 引擎接口和实现
│   ├── crypto/              DPAPI 加解密
│   ├── storage/             存储管理 + SQLite 索引
│   └── icon/                托盘图标生成
├── ui/                      用户界面
│   ├── tray/                系统托盘（systray）
│   ├── server/              Web 管理服务器 + HTML 模板
│   ├── panel/               保留：原生配置面板
│   └── viewer/              保留：证据浏览器
└── Arch.md                  架构决策记录
```

## 编码约定

### Go 版本
- Go 1.21+
- 优先使用 `golang.org/x/sys/windows` 而非 `syscall`

### 包导入顺序
```
1. 标准库
2. 第三方库
3. 内部包
```

### Windows API 调用
- 统一使用 `syscall.NewLazyDLL` + `NewProc` 方式
- 不要用 cgo 调用 Win32 API
- 函数名保持 Win32 原始名称

```go
// 正确
var user32 = syscall.NewLazyDLL("user32.dll")
var procEnumWindows = user32.NewProc("EnumWindows")
```

### 错误处理
- Windows API 返回 0 表示失败，用 `syscall.GetLastError()` 取错误码
- 日志用 `log.Printf` 中文描述
- 非关键错误不 panic，只 log 后继续

### 配置
- 所有可配项在 `internal/config/config.go` 的 `Config` 结构体中定义
- 默认值在 `DefaultConfig` 中设定
- YAML 标签使用 snake_case

### 安全
- 证据文件必须通过 `internal/crypto` 加密
- 管理面板只监听 `127.0.0.1`，不对外暴露
- 快捷键和 API 调用不做认证（纯本地应用）

## 模块开发注意

### 新增目标应用
1. 在 `config.go` 的 `DefaultConfig.Targets` 中添加默认项
2. 在 `config.yaml` 中添加对应条目
3. 验证 `chrome-cleaner\windows_class_detector` 中的类名

### 修改触发逻辑
- 标题检测 (L1) 2秒间隔，用 `time.NewTicker`
- OCR (L3) 间隔在配置中设定，默认 10-30 秒
- 热键消息循环必须 `runtime.LockOSThread()`

### 管理面板 API
- 路径以 `/api/` 开头
- 返回 JSON，Content-Type: application/json
- 新增 API 端点要在 `server.go` 中注册路由

## 测试
- 暂未配置测试框架，后续补充

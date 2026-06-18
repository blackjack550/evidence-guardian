# 证据卫士 一键打包脚本
# 作用: 下载 FFmpeg 和 Tesseract 便携版到 bin 目录，形成完整发布包

param(
    [string]$OutputDir = ".\dist\证据卫士_v0.1.0"
)

$ErrorActionPreference = "Stop"
$Proxy = "socks5://10.5.254.8:9999"
$env:HTTP_PROXY = $Proxy
$env:HTTPS_PROXY = $Proxy

Write-Host "=== 证据卫士 打包脚本 ===" -ForegroundColor Cyan
Write-Host "输出目录: $OutputDir`n"

# 1. 编译主程序
Write-Host "[1/4] 编译主程序..." -ForegroundColor Yellow
$env:GOPROXY = "https://goproxy.cn,direct"
go build -ldflags="-H=windowsgui" -o "$OutputDir\evidence-guardian.exe" .

# 2. 下载 FFmpeg (静态单文件)
Write-Host "[2/4] 下载 FFmpeg..." -ForegroundColor Yellow
$ffmpegUrl = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"
$ffmpegZip = "$env:TEMP\ffmpeg.zip"
Invoke-WebRequest -Uri $ffmpegUrl -OutFile $ffmpegZip -UseBasicParsing -TimeoutSec 120
Expand-Archive -Path $ffmpegZip -DestinationPath "$env:TEMP\ffmpeg" -Force
Copy-Item "$env:TEMP\ffmpeg\ffmpeg-master-latest-win64-gpl\bin\ffmpeg.exe" "$OutputDir\ffmpeg.exe"

# 3. 下载 Tesseract 便携版 + 中文语言包
Write-Host "[3/4] 下载 Tesseract 便携版..." -ForegroundColor Yellow
$tessUrl = "https://github.com/UB-Mannheim/tesseract/releases/download/v5.4.0.20240606/tesseract-ocr-w64-portable-5.4.0.20240606.zip"
$tessZip = "$env:TEMP\tesseract.zip"
Invoke-WebRequest -Uri $tessUrl -OutFile $tessZip -UseBasicParsing -TimeoutSec 120
Expand-Archive -Path $tessZip -DestinationPath "$OutputDir\tesseract" -Force

# 下载中文语言包
Write-Host "     下载中文语言包..." -ForegroundColor Yellow
$langUrl = "https://github.com/tesseract-ocr/tessdata_fast/raw/main/chi_sim.traineddata"
Invoke-WebRequest -Uri $langUrl -OutFile "$OutputDir\tesseract\tessdata\chi_sim.traineddata" -UseBasicParsing -TimeoutSec 60

# 4. 复制默认配置和说明
Write-Host "[4/4] 生成配置和说明..." -ForegroundColor Yellow
Copy-Item "config.yaml" "$OutputDir\" -ErrorAction SilentlyContinue

@"
证据卫士 v0.1.0
劳动者权益保护取证系统

使用说明:
1. 双击 evidence-guardian.exe 启动（后台运行，无窗口）
2. 浏览器打开 http://127.0.0.1:58080 进入管理面板
3. 如需 Chrome/Edge 浏览器检测，使用桌面上的专用快捷方式启动浏览器
4. 如需视频录制功能，确认 ffmpeg.exe 在目录中

目录结构:
├── evidence-guardian.exe    主程序
├── ffmpeg.exe               视频编码（可选）
├── tesseract/               OCR 引擎
│   ├── tesseract.exe
│   └── tessdata/            语言数据
├── config.yaml              配置文件
└── README.txt               本文件
"@ | Out-File -FilePath "$OutputDir\README.txt" -Encoding UTF8

Write-Host "`n=== 打包完成 ===" -ForegroundColor Green
Write-Host "输出: $OutputDir"
Write-Host "总大小: $(Get-ChildItem -Recurse $OutputDir | Measure-Object -Property Length -Sum | ForEach-Object { '{0:N0} KB' -f ($_.Sum / 1KB) })"

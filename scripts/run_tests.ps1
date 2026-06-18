# 证据卫士 自动化测试脚本
# 用法: powershell -File scripts\run_tests.ps1

$ErrorActionPreference = "Continue"
$env:GOPROXY = "https://goproxy.cn,direct"
$passed = 0
$failed = 0

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  证据卫士 自动化测试套件" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# ─── 1. 单元测试 ────────────────────────────────
Write-Host "━━━ [1/4] 运行单元测试 ━━━" -ForegroundColor Yellow

$unitTests = @(
    @{Name="crypto"; Pkg="./internal/crypto/..."},
    @{Name="config"; Pkg="./internal/config/..."},
    @{Name="trigger"; Pkg="./internal/trigger/..."},
    @{Name="storage"; Pkg="./internal/storage/..."}
)

foreach ($t in $unitTests) {
    Write-Host "  Testing $($t.Name)..." -NoNewline
    $result = go test -count=1 -timeout 30s $t.Pkg 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host " PASS" -ForegroundColor Green
        $passed++
    } else {
        Write-Host " FAIL" -ForegroundColor Red
        Write-Host "  $result" -ForegroundColor Red
        $failed++
    }
}

Write-Host ""

# ─── 2. 编译检查 ────────────────────────────────
Write-Host "━━━ [2/4] 编译检查 ━━━" -ForegroundColor Yellow
Write-Host "  Building GUI binary..." -NoNewline
go build -ldflags="-H=windowsgui" -o "$env:TEMP\ev_test_build.exe" . 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host " PASS" -ForegroundColor Green
    Remove-Item "$env:TEMP\ev_test_build.exe" -Force -ErrorAction SilentlyContinue
    $passed++
} else {
    Write-Host " FAIL" -ForegroundColor Red
    $failed++
}

Write-Host ""

# ─── 3. 集成测试（需启动程序） ───────────────────
Write-Host "━━━ [3/4] 集成测试 ━━━" -ForegroundColor Yellow
Write-Host "  Starting evidence-guardian..." -NoNewline

# Kill existing and start fresh
Get-Process -Name "evidence-guardian" -ErrorAction SilentlyContinue | Stop-Process -Force
Remove-Item -Recurse -Force "D:\openproject\evidence-guardian\bin\evidence" -ErrorAction SilentlyContinue
Remove-Item -Force "D:\openproject\evidence-guardian\bin\evidence-guardian.log" -ErrorAction SilentlyContinue
Remove-Item -Force "D:\openproject\evidence-guardian\bin\config.yaml" -ErrorAction SilentlyContinue

go build -ldflags="-H=windowsgui" -o "D:\openproject\evidence-guardian\bin\evidence-guardian.exe" . 2>&1
Start-Process -FilePath "D:\openproject\evidence-guardian\bin\evidence-guardian.exe" -WindowStyle Hidden
Start-Sleep -Seconds 6

if ((Get-Process -Name "evidence-guardian" -ErrorAction SilentlyContinue).Count -gt 0) {
    Write-Host " PASS" -ForegroundColor Green
    $passed++
} else {
    Write-Host " FAIL" -ForegroundColor Red
    $failed++
}

# ─── 4. API 测试 ────────────────────────────────
Write-Host "  Testing API endpoints..."

$apiTests = @(
    @{Name="GET /api/status"; Method="GET"; Url="http://127.0.0.1:58080/api/status"; Expect=200},
    @{Name="GET /api/config"; Method="GET"; Url="http://127.0.0.1:58080/api/config"; Expect=200},
    @{Name="POST /api/trigger"; Method="POST"; Url="http://127.0.0.1:58080/api/trigger"; Expect=200},
    @{Name="POST /api/config"; Method="POST"; Url="http://127.0.0.1:58080/api/config"; Body='{"capture_mode":"desktop"}'; Expect=200}
)

foreach ($t in $apiTests) {
    Write-Host "    $($t.Name)..." -NoNewline
    try {
        if ($t.Body) {
            $r = Invoke-WebRequest -Uri $t.Url -Method $t.Method -Body $t.Body -ContentType "application/json" -UseBasicParsing -TimeoutSec 10
        } else {
            $r = Invoke-WebRequest -Uri $t.Url -Method $t.Method -UseBasicParsing -TimeoutSec 10
        }
        if ($r.StatusCode -eq $t.Expect) {
            Write-Host " PASS" -ForegroundColor Green
            $passed++
        } else {
            Write-Host " FAIL (Status: $($r.StatusCode))" -ForegroundColor Red
            $failed++
        }
    } catch {
        Write-Host " FAIL ($($_.Exception.Message))" -ForegroundColor Red
        $failed++
    }
}

# Wait for screenshot and verify evidence
Write-Host "  Verifying evidence capture..." -NoNewline
Start-Sleep -Seconds 4
try {
    $ev = Invoke-WebRequest -Uri "http://127.0.0.1:58080/api/evidence" -UseBasicParsing -TimeoutSec 10
    $content = $ev.Content | ConvertFrom-Json
    if ($content.Count -gt 0 -or $content.value.Count -gt 0) {
        Write-Host " PASS" -ForegroundColor Green
        $passed++
    } else {
        Write-Host " FAIL (no evidence)" -ForegroundColor Red
        $failed++
    }
} catch {
    Write-Host " FAIL ($($_.Exception.Message))" -ForegroundColor Red
    $failed++
}

# Cleanup
Get-Process -Name "evidence-guardian" -ErrorAction SilentlyContinue | Stop-Process -Force
Remove-Item -Recurse -Force "D:\openproject\evidence-guardian\bin\evidence" -ErrorAction SilentlyContinue
Remove-Item -Force "D:\openproject\evidence-guardian\bin\evidence-guardian.log" -ErrorAction SilentlyContinue
Remove-Item -Force "D:\openproject\evidence-guardian\bin\config.yaml" -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  测试完成" -ForegroundColor Cyan
Write-Host "  通过: $passed  失败: $failed" -ForegroundColor $(if ($failed -eq 0) {"Green"} else {"Red"})
Write-Host "========================================" -ForegroundColor Cyan

if ($failed -gt 0) { exit 1 } else { exit 0 }

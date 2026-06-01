; 证据卫士 安装脚本 (InnoSetup)
; 使用方法:
;   1. 先运行 bundle.ps1 生成 dist 目录
;   2. 用 InnoSetup 打开此文件编译

[Setup]
AppName=证据卫士
AppVersion=0.1.0
AppPublisher=Evidence Guardian
DefaultDirName={autopf}\证据卫士
DefaultGroupName=证据卫士
UninstallDisplayIcon={app}\evidence-guardian.exe
Compression=lzma2
SolidCompression=yes
OutputDir=..\dist
OutputBaseFilename=证据卫士_v0.1.0_安装包

[Files]
Source: "..\dist\证据卫士_v0.1.0\evidence-guardian.exe"; DestDir: "{app}"
Source: "..\dist\证据卫士_v0.1.0\ffmpeg.exe"; DestDir: "{app}"
Source: "..\dist\证据卫士_v0.1.0\tesseract\*"; DestDir: "{app}\tesseract"; Flags: recursesubdirs
Source: "..\dist\证据卫士_v0.1.0\config.yaml"; DestDir: "{app}"

[Icons]
Name: "{group}\证据卫士"; Filename: "{app}\evidence-guardian.exe"
Name: "{group}\证据卫士 管理面板"; Filename: "http://127.0.0.1:58080"
Name: "{group}\卸载证据卫士"; Filename: "{uninstallexe}"
Name: "{commondesktop}\证据卫士"; Filename: "{app}\evidence-guardian.exe"
Name: "{commondesktop}\Chrome（证据卫士调试模式）"; Filename: "{code:GetChromePath}"; Parameters: "--remote-debugging-port=9222"

[Run]
Filename: "{app}\evidence-guardian.exe"; Flags: nowait; Description: "启动证据卫士"

[Code]
function GetChromePath(Param: string): string;
begin
  if RegQueryStringValue(HKLM, 'SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe', '', Result) then
    exit;
  if RegQueryStringValue(HKCU, 'SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe', '', Result) then
    exit;
  Result := 'C:\Program Files\Google\Chrome\Application\chrome.exe';
end;

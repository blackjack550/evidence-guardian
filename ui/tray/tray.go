package tray

import (
	"fmt"
	"log"
	"os/exec"

	"evidence-guardian/internal/config"
	"evidence-guardian/internal/storage"
	"evidence-guardian/internal/trigger"
	"evidence-guardian/ui/server"
	"github.com/getlantern/systray"
)

func Run(cfg *config.Config, engine *trigger.Engine, store *storage.Manager, srv *server.Server) {
	systray.Run(
		func() { onReady(cfg, engine, store, srv) },
		func() { onExit() },
	)
}

func onReady(cfg *config.Config, engine *trigger.Engine, store *storage.Manager, srv *server.Server) {
	systray.SetTitle("证据卫士")
	systray.SetTooltip("劳动者权益保护取证系统")

	mCapture := systray.AddMenuItem("立即取证", "手动触发证据采集")
	mPanel := systray.AddMenuItem("打开管理面板", "打开 Web 管理页面")
	systray.AddSeparator()
	mAutoStart := systray.AddMenuItemCheckbox("开机自启", "随Windows开机自动启动", cfg.AutoStart)
	mQuit := systray.AddMenuItem("退出", "退出程序")

	go func() {
		for {
			select {
			case <-mCapture.ClickedCh:
				engine.ManualTrigger("tray_menu")
			case <-mPanel.ClickedCh:
				openBrowser(fmt.Sprintf("http://127.0.0.1:%d", srv.Port()))
			case <-mAutoStart.ClickedCh:
				toggleAutoStart(mAutoStart)
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	log.Println("证据卫士已退出")
}

func ShowNotify(title, message string) {
	exec.Command("powershell",
		"-NoProfile", "-Command",
		fmt.Sprintf(`
			[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
			$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
			$textNodes = $template.GetElementsByTagName("text")
			$textNodes.Item(0).AppendChild($template.CreateTextNode('%s')) > $null
			$textNodes.Item(1).AppendChild($template.CreateTextNode('%s')) > $null
			$toast = [Windows.UI.Notifications.ToastNotification]::new($template)
			[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier().Show($toast)
		`, title, message),
	).Start()
}

func toggleAutoStart(item *systray.MenuItem) {
	if item.Checked() {
		item.Uncheck()
	} else {
		item.Check()
	}
}

func openBrowser(url string) {
	exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

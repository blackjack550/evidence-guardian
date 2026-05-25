package tray

import (
	"log"

	"evidence-guardian/internal/config"
	"evidence-guardian/internal/storage"
	"evidence-guardian/internal/trigger"
	"github.com/getlantern/systray"
)

func Run(cfg *config.Config, engine *trigger.Engine, store *storage.Manager) {
	systray.Run(
		func() { onReady(cfg, engine, store) },
		func() { onExit() },
	)
}

func onReady(cfg *config.Config, engine *trigger.Engine, store *storage.Manager) {
	systray.SetTitle("证据卫士")
	systray.SetTooltip("劳动者权益保护取证系统")

	mCapture := systray.AddMenuItem("立即取证", "手动触发证据采集")
	systray.AddSeparator()
	mPanel := systray.AddMenuItem("打开配置面板", "配置关键词、应用、存储等")
	mViewer := systray.AddMenuItem("查看证据记录", "浏览已采集的证据")
	systray.AddSeparator()
	mAutoStart := systray.AddMenuItemCheckbox("开机自启", "随Windows开机自动启动", cfg.AutoStart)
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出程序")

	go func() {
		for {
			select {
			case <-mCapture.ClickedCh:
				log.Println("用户手动取证")
			case <-mPanel.ClickedCh:
				showPanel(cfg)
			case <-mViewer.ClickedCh:
				showViewer(store)
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

func showPanel(cfg *config.Config) {
	// TODO: open configuration panel window
}

func showViewer(store *storage.Manager) {
	// TODO: open evidence viewer window
}

func toggleAutoStart(item *systray.MenuItem) {
	if item.Checked() {
		item.Uncheck()
		removeAutoStart()
	} else {
		item.Check()
		installAutoStart()
	}
}

func installAutoStart() {
	// TODO: write to HKCU\...\Run
}

func removeAutoStart() {
	// TODO: remove from HKCU\...\Run
}

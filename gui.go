package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jerr3n/everblue/util"
	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//go:embed assets/everblue.svg
var logoSVG string

// detachDelay is how long Studio (or its installer) has to stay up before the progress dialog gets out of its way
const detachDelay = 5 * time.Second

// svgPixmap renders svg at size logical pixels, doubled so it stays sharp on HiDPI screens
func svgPixmap(svg string, size int) *qt.QPixmap {
	svg = strings.Replace(svg, "<svg ", fmt.Sprintf(`<svg width="%d" height="%d" `, size*2, size*2), 1)
	pix := qt.NewQPixmap()
	pix.LoadFromData4([]byte(svg), "SVG")
	pix.SetDevicePixelRatio(2)
	return pix
}

// paneWriter forwards zap output into the log pane
type paneWriter struct {
	pane *qt.QPlainTextEdit
}

func (w paneWriter) Write(p []byte) (int, error) {
	line := strings.TrimRight(string(p), "\n")
	mainthread.Start(func() { w.pane.AppendPlainText(line) })
	return len(p), nil
}

func (w paneWriter) Sync() error { return nil }

// setName gives a widget the object name the stylesheet styles it by
func setName(o *qt.QObject, name string) {
	view := qt.NewQAnyStringView3(name)
	o.SetObjectName(*view)
}

// label makes a label styled by its object name
func label(text, style string) *qt.QLabel {
	l := qt.NewQLabel3(text)
	if style != "" {
		setName(l.QObject, style)
	}
	return l
}

func button(text, style string) *qt.QPushButton {
	b := qt.NewQPushButton3(text)
	if style != "" {
		setName(b.QObject, style)
	}
	b.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	b.SetAutoDefault(false)
	return b
}

// card is a rounded panel with a vertical layout inside
func card(style string) (*qt.QFrame, *qt.QVBoxLayout) {
	frame := qt.NewQFrame2()
	setName(frame.QObject, style)
	layout := qt.NewQVBoxLayout2()
	layout.SetContentsMargins(22, 20, 22, 20)
	layout.SetSpacing(10)
	frame.SetLayout(layout.QLayout)
	return frame, layout
}

// page is one of the sidebar's destinations, with its title and what it's for
func page(title, subtitle string) (*qt.QWidget, *qt.QVBoxLayout) {
	w := qt.NewQWidget2()
	setName(w.QObject, "page")
	layout := qt.NewQVBoxLayout2()
	layout.SetContentsMargins(36, 32, 36, 28)
	layout.SetSpacing(16)
	layout.AddWidget(label(title, "h1").QWidget)
	if subtitle != "" {
		sub := label(subtitle, "muted")
		sub.SetWordWrap(true)
		layout.AddWidget(sub.QWidget)
	}
	layout.AddSpacing(4)
	w.SetLayout(layout.QLayout)
	return w, layout
}

// scrolled lets a page scroll rather than squeeze its cards when the window is short
func scrolled(w *qt.QWidget) *qt.QWidget {
	scroll := qt.NewQScrollArea2()
	scroll.SetWidget(w)
	scroll.SetWidgetResizable(true)
	scroll.SetFrameShape(qt.QFrame__NoFrame)
	return scroll.QWidget
}

func row(widgets ...*qt.QWidget) *qt.QHBoxLayout {
	layout := qt.NewQHBoxLayout2()
	layout.SetSpacing(10)
	for _, w := range widgets {
		if w == nil {
			layout.AddStretch()
		} else {
			layout.AddWidget(w)
		}
	}
	return layout
}

type progressDialog struct {
	*qt.QDialog
	status *qt.QLabel
	info   *qt.QLabel
	bar    *qt.QProgressBar
	logo   *spinningLogo
	cancel context.CancelFunc
}

func newProgressDialog(parent *qt.QWidget, color accent, still bool) *progressDialog {
	d := &progressDialog{QDialog: qt.NewQDialog(parent)}
	d.SetWindowTitle("Everblue")
	d.SetWindowModality(qt.WindowModal)
	d.SetMinimumWidth(480)

	d.logo = newSpinningLogo(d.QWidget, color, still)
	d.logo.SetFixedSize2(72, 72)
	d.status = label("", "h2")
	d.info = label("", "muted")
	d.bar = qt.NewQProgressBar2()
	d.bar.SetFixedHeight(10)

	text := qt.NewQVBoxLayout2()
	text.SetSpacing(6)
	text.AddStretch()
	text.AddWidget(d.status.QWidget)
	text.AddWidget(d.info.QWidget)
	text.AddSpacing(6)
	text.AddWidget(d.bar.QWidget)
	text.AddStretch()
	top := qt.NewQHBoxLayout2()
	top.SetSpacing(18)
	top.AddWidget(d.logo.QWidget)
	top.AddLayout2(text.QLayout, 1)

	cancel := button("Cancel", "")
	cancel.OnClicked(d.Reject)
	// rejected covers both the Cancel button and closing the window
	d.OnRejected(func() {
		if d.cancel != nil {
			d.cancel()
		}
	})

	layout := qt.NewQVBoxLayout2()
	layout.SetContentsMargins(24, 22, 24, 20)
	layout.SetSpacing(16)
	layout.AddLayout(top.QLayout)
	layout.AddLayout(row(nil, cancel.QWidget).QLayout)
	d.SetLayout(layout.QLayout)
	return d
}

type gui struct {
	home     string
	logger   *zap.Logger
	latest   string
	busy     bool
	settings settings

	app         *qt.QApplication
	manager     *qt.QMainWindow
	pages       *qt.QStackedWidget
	nav         []*qt.QPushButton
	logo        *spinningLogo
	footer      *qt.QLabel
	logPane     *qt.QPlainTextEdit
	progress    *progressDialog
	runAction   *qt.QAction
	checkAction *qt.QAction

	// Home
	heroTitle     *qt.QLabel
	heroDetail    *qt.QLabel
	statusPill    *qt.QLabel
	playButton    *qt.QPushButton
	installButton *qt.QPushButton
	latestValue   *qt.QLabel
	channelValue  *qt.QLabel
	displayValue  *qt.QLabel
	baseValue     *qt.QLabel

	// Versions
	channelBox     *qt.QComboBox
	channelLatest  *qt.QLabel
	installLatest  *qt.QPushButton
	versionList    *qt.QListWidget
	versionRows    []installedStudio
	activateButton *qt.QPushButton
	deleteButton   *qt.QPushButton
	hashEdit       *qt.QLineEdit
	hashInstall    *qt.QPushButton

	// Fast flags
	flagTable   *qt.QTableWidget
	flagSearch  *qt.QLineEdit
	flagCount   *qt.QLabel
	loadingFlag bool

	// Settings
	swatches      map[string]*qt.QPushButton
	rosemaryValue *qt.QLabel

	// Diagnostics
	diagResults *qt.QVBoxLayout
	diagButton  *qt.QPushButton

	// quitWhenDone ends the app once a task finishes, for the bare launcher
	quitWhenDone bool
}

const (
	pageHome = iota
	pageVersions
	pageFlags
	pageSettings
	pageDiagnostics
	pageLogs
)

// runConfig opens the main window
func runConfig(home string, logger *zap.Logger) {
	g := newGUI(home, logger)
	g.showManager()
	qt.QApplication_Exec()
}

// runLauncher launches Studio with just the progress dialog and exits when Studio does.
// Without Studio installed there is nothing to launch, so it opens the main window instead.
func runLauncher(home string, logger *zap.Logger) {
	g := newGUI(home, logger)
	if studioInstalled(home) && installedVersion(home) != "" {
		g.quitWhenDone = true
		// cancelling the display system choice launches nothing, and there's no window to stay open for
		if !g.launch() {
			return
		}
	} else {
		g.showManager()
	}
	qt.QApplication_Exec()
}

func (g *gui) showManager() {
	g.refresh()
	g.checkForUpdates()
	g.manager.Show()
}

func newGUI(home string, logger *zap.Logger) *gui {
	qt.QCoreApplication_SetApplicationName("Everblue")
	qt.QGuiApplication_SetDesktopFileName("everblue")
	g := &gui{home: home, settings: loadSettings(home, logger)}
	g.app = qt.NewQApplication(os.Args)
	qt.QGuiApplication_SetWindowIcon(qt.NewQIcon2(svgPixmap(logoSVG, 128)))
	applyTheme(g.app, g.home, g.settings.Accent)

	g.logPane = qt.NewQPlainTextEdit2()
	g.logger = zap.New(zapcore.NewTee(logger.Core(), zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		paneWriter{g.logPane},
		zap.DebugLevel,
	)))
	g.buildManager()
	g.progress = newProgressDialog(g.manager.QWidget, accentFor(g.settings.Accent), g.settings.StillLogo)
	return g
}

func (g *gui) setStatus(text string) {
	g.footer.SetText(text)
}

func (g *gui) openFolder(path string) func() {
	return func() { qt.QDesktopServices_OpenUrl(qt.QUrl_FromLocalFile(path)) }
}

func (g *gui) buildManager() {
	g.manager = qt.NewQMainWindow2()
	g.manager.SetWindowTitle("Everblue")
	g.manager.Resize(1040, 680)
	g.manager.SetMinimumSize2(900, 600)

	g.pages = qt.NewQStackedWidget2()
	g.pages.AddWidget(scrolled(g.buildHome()))
	g.pages.AddWidget(scrolled(g.buildVersions()))
	g.pages.AddWidget(g.buildFlags())
	g.pages.AddWidget(scrolled(g.buildSettings()))
	g.pages.AddWidget(scrolled(g.buildDiagnostics()))
	g.pages.AddWidget(g.buildLogs())

	body := qt.NewQHBoxLayout2()
	body.SetContentsMargins(0, 0, 0, 0)
	body.SetSpacing(0)
	body.AddWidget(g.buildSidebar())
	body.AddWidget2(g.pages.QWidget, 1)
	central := qt.NewQWidget2()
	central.SetLayout(body.QLayout)
	g.manager.SetCentralWidget(central)

	// The logo sits over the bottom right corner of the home page, out of the way of everything else
	const logoSize = 190
	g.logo = newSpinningLogo(central, accentFor(g.settings.Accent), g.settings.StillLogo)
	g.logo.SetFixedSize2(logoSize, logoSize)
	central.OnResizeEvent(func(super func(*qt.QResizeEvent), e *qt.QResizeEvent) {
		super(e)
		g.logo.Move(central.Width()-logoSize-28, central.Height()-logoSize-22)
		g.logo.Raise()
	})

	// Shortcuts without a menu bar to show them in
	shortcut := func(keys string, fn func()) *qt.QAction {
		action := qt.NewQAction2("")
		action.SetShortcut(qt.NewQKeySequence2(keys))
		action.OnTriggered(fn)
		g.manager.AddAction(action)
		return action
	}
	g.runAction = shortcut("Ctrl+R", func() { g.launch() })
	shortcut("Ctrl+U", g.install)
	g.checkAction = shortcut("F5", g.checkForUpdates)
	shortcut("Ctrl+D", func() { g.showPage(pageDiagnostics); g.diagnose() })
	shortcut("Ctrl+L", func() { g.showPage(pageLogs) })
	quit := qt.NewQAction2("")
	quit.SetShortcut(qt.NewQKeySequence6(qt.QKeySequence__Quit))
	quit.OnTriggered(func() { g.manager.Close() })
	g.manager.AddAction(quit)

	g.showPage(pageHome)
}

func (g *gui) buildSidebar() *qt.QWidget {
	sidebar := qt.NewQWidget2()
	setName(sidebar.QObject, "sidebar")
	sidebar.SetAttribute(qt.WA_StyledBackground)
	sidebar.SetFixedWidth(232)
	layout := qt.NewQVBoxLayout2()
	layout.SetContentsMargins(16, 22, 16, 18)
	layout.SetSpacing(4)

	mark := qt.NewQLabel2()
	mark.SetPixmap(svgPixmap(logoSVG, 34))
	brandText := qt.NewQVBoxLayout2()
	brandText.SetSpacing(0)
	brandText.AddWidget(label("Everblue", "brand").QWidget)
	brandText.AddWidget(label("STUDIO LAUNCHER", "brandSub").QWidget)
	brand := qt.NewQHBoxLayout2()
	brand.SetContentsMargins(6, 0, 0, 0)
	brand.SetSpacing(12)
	brand.AddWidget(mark.QWidget)
	brand.AddLayout2(brandText.QLayout, 1)
	layout.AddLayout(brand.QLayout)
	layout.AddSpacing(26)

	group := qt.NewQButtonGroup()
	group.SetExclusive(true)
	for i, name := range []string{"Home", "Versions", "Fast Flags", "Settings", "Diagnostics", "Logs"} {
		b := button(name, "nav")
		b.SetCheckable(true)
		index := i
		b.OnClicked(func() { g.showPage(index) })
		group.AddButton(b.QAbstractButton)
		g.nav = append(g.nav, b)
		layout.AddWidget(b.QWidget)
	}
	layout.AddStretch()
	g.footer = label("Ready", "sidebarFooter")
	g.footer.SetWordWrap(true)
	layout.AddWidget(g.footer.QWidget)
	sidebar.SetLayout(layout.QLayout)
	return sidebar
}

func (g *gui) showPage(index int) {
	g.pages.SetCurrentIndex(index)
	g.nav[index].SetChecked(true)
	if g.logo != nil {
		g.logo.SetVisible(index == pageHome)
	}
	switch index {
	case pageVersions:
		g.refreshVersions()
	case pageDiagnostics:
		if g.diagResults.Count() == 0 {
			g.diagnose()
		}
	}
}

// --- Home ---

func (g *gui) buildHome() *qt.QWidget {
	w, layout := page("Roblox Studio", "")
	// the page title doubles as the hero's, so drop the generic one
	layout.ItemAt(0).Widget().Hide()

	hero, heroLayout := card("hero")
	heroLayout.SetContentsMargins(32, 30, 32, 30)
	heroLayout.SetSpacing(8)
	g.statusPill = label("", "pillGood")
	heroLayout.AddLayout(row(label("ROBLOX STUDIO", "caption").QWidget, nil, g.statusPill.QWidget).QLayout)
	g.heroTitle = label("", "h1")
	heroLayout.AddWidget(g.heroTitle.QWidget)
	g.heroDetail = label("", "muted")
	heroLayout.AddWidget(g.heroDetail.QWidget)
	heroLayout.AddSpacing(18)
	g.playButton = button("▶  Play", "play")
	g.playButton.OnClicked(func() { g.launch() })
	g.installButton = button("Install", "")
	g.installButton.SetMinimumHeight(48)
	g.installButton.OnClicked(g.install)
	heroLayout.AddLayout(row(g.playButton.QWidget, g.installButton.QWidget, nil).QLayout)
	layout.AddWidget(hero.QWidget)

	info := qt.NewQGridLayout2()
	info.SetSpacing(14)
	tile := func(caption string, r, c int) *qt.QLabel {
		frame, l := card("card")
		l.SetSpacing(4)
		l.AddWidget(label(caption, "caption").QWidget)
		value := label("", "value")
		value.SetTextInteractionFlags(qt.TextSelectableByMouse)
		l.AddWidget(value.QWidget)
		info.AddWidget2(frame.QWidget, r, c)
		return value
	}
	g.latestValue = tile("LATEST ON CHANNEL", 0, 0)
	g.channelValue = tile("RELEASE CHANNEL", 0, 1)
	g.displayValue = tile("DISPLAY SYSTEM", 1, 0)
	g.baseValue = tile("ROSEMARY BASE SYSTEM", 1, 1)
	infoWidget := qt.NewQWidget2()
	infoWidget.SetLayout(info.QLayout)
	// leave the right of the page to the logo
	infoRow := qt.NewQHBoxLayout2()
	infoRow.AddWidget2(infoWidget, 3)
	infoRow.AddStretchWithStretch(1)
	layout.AddLayout(infoRow.QLayout)
	layout.AddStretch()
	return w
}

// --- Versions ---

func (g *gui) buildVersions() *qt.QWidget {
	w, layout := page("Versions", "Keep several versions of Studio, switch between them, and follow a release channel.")

	channelCard, channelLayout := card("card")
	channelLayout.AddWidget(label("Release channel", "h2").QWidget)
	channelNote := label("LIVE is what everyone gets. Most other channels are private to Roblox and refuse to answer.", "faint")
	channelNote.SetWordWrap(true)
	channelLayout.AddWidget(channelNote.QWidget)
	g.channelBox = qt.NewQComboBox2()
	g.channelBox.SetEditable(true)
	g.channelBox.AddItems([]string{util.LiveChannel, "zcanary", "zintegration", "zstudiobeta"})
	g.channelBox.SetCurrentText(g.settings.channel())
	g.channelBox.SetMinimumWidth(220)
	useChannel := button("Use Channel", "")
	useChannel.OnClicked(func() {
		channel := strings.TrimSpace(g.channelBox.CurrentText())
		if channel == "" || strings.EqualFold(channel, util.LiveChannel) {
			channel = ""
		}
		g.settings.Channel = channel
		g.settings.save(g.home, g.logger)
		g.checkForUpdates()
	})
	channelLayout.AddLayout(row(g.channelBox.QWidget, useChannel.QWidget, nil).QLayout)
	g.channelLatest = label("", "muted")
	g.installLatest = button("Install Latest", "primary")
	g.installLatest.OnClicked(func() { g.installHash(g.latest) })
	channelLayout.AddLayout(row(g.channelLatest.QWidget, nil, g.installLatest.QWidget).QLayout)
	layout.AddWidget(channelCard.QWidget)

	installedCard, installedLayout := card("card")
	installedLayout.AddWidget(label("Installed", "h2").QWidget)
	g.versionList = qt.NewQListWidget2()
	g.versionList.SetMinimumHeight(170)
	g.versionList.OnCurrentRowChanged(func(int) { g.refreshVersionButtons() })
	installedLayout.AddWidget(g.versionList.QWidget)
	g.activateButton = button("Switch To", "primary")
	g.activateButton.OnClicked(g.activateSelected)
	g.deleteButton = button("Delete", "danger")
	g.deleteButton.OnClicked(g.deleteSelected)
	openStore := button("Open Folder", "")
	openStore.OnClicked(func() {
		os.MkdirAll(versionStore(g.home), 0o755)
		g.openFolder(versionStore(g.home))()
	})
	installedLayout.AddLayout(row(g.activateButton.QWidget, g.deleteButton.QWidget, nil, openStore.QWidget).QLayout)
	layout.AddWidget2(installedCard.QWidget, 1)

	hashCard, hashLayout := card("card")
	hashLayout.AddWidget(label("Install a specific version", "h2").QWidget)
	hashNote := label("Downloads the whole of that Studio (around 600 MB) from the release channel above.", "faint")
	hashNote.SetWordWrap(true)
	hashLayout.AddWidget(hashNote.QWidget)
	g.hashEdit = qt.NewQLineEdit2()
	g.hashEdit.SetPlaceholderText("version-0123456789abcdef")
	g.hashInstall = button("Install", "")
	g.hashInstall.OnClicked(func() { g.installHash(strings.TrimSpace(g.hashEdit.Text())) })
	g.hashEdit.OnReturnPressed(func() { g.installHash(strings.TrimSpace(g.hashEdit.Text())) })
	hashLayout.AddLayout(row(g.hashEdit.QWidget, g.hashInstall.QWidget).QLayout)
	layout.AddWidget(hashCard.QWidget)
	return w
}

func (g *gui) refreshVersions() {
	selected := ""
	if i := g.versionList.CurrentRow(); i >= 0 && i < len(g.versionRows) {
		selected = g.versionRows[i].Hash
	}
	g.versionRows = listVersions(g.home)
	g.versionList.Clear()
	for i, v := range g.versionRows {
		version := v.Version
		if version == "" {
			version = "Unknown version"
		}
		channel := v.Channel
		if channel == "" {
			channel = util.LiveChannel
		}
		text := fmt.Sprintf("%s\n%s  ·  %s", version, v.Hash, channel)
		if v.Active {
			text = fmt.Sprintf("%s   —   ACTIVE\n%s  ·  %s", version, v.Hash, channel)
		}
		g.versionList.AddItem(text)
		if v.Hash == selected {
			g.versionList.SetCurrentRow(i)
		}
	}
	if len(g.versionRows) == 0 {
		g.versionList.AddItem("No versions installed yet")
	}
	g.refreshVersionButtons()
}

func (g *gui) selectedVersion() *installedStudio {
	i := g.versionList.CurrentRow()
	if i < 0 || i >= len(g.versionRows) {
		return nil
	}
	return &g.versionRows[i]
}

func (g *gui) refreshVersionButtons() {
	v := g.selectedVersion()
	g.activateButton.SetEnabled(!g.busy && v != nil && !v.Active)
	g.deleteButton.SetEnabled(!g.busy && v != nil && !v.Active)
	g.installLatest.SetEnabled(!g.busy && g.latest != "" && g.latest != installedVersion(g.home))
	g.hashInstall.SetEnabled(!g.busy)
}

func (g *gui) activateSelected() {
	v := g.selectedVersion()
	if v == nil {
		return
	}
	hash := v.Hash
	g.runTask("Switching versions", "Switched versions", func(ctx context.Context, progress util.ProgressFunc, detach func()) error {
		progress(hash, 0, -1)
		return activateVersion(g.home, g.logger, hash)
	})
}

func (g *gui) deleteSelected() {
	v := g.selectedVersion()
	if v == nil {
		return
	}
	answer := qt.QMessageBox_Question5(g.manager.QWidget, "Delete Version",
		fmt.Sprintf("Delete Studio %s (%s)?", v.Version, v.Hash), qt.QMessageBox__Yes|qt.QMessageBox__Cancel)
	if answer != qt.QMessageBox__Yes {
		return
	}
	if err := deleteVersion(g.home, g.logger, v.Hash); err != nil {
		qt.QMessageBox_Critical(g.manager.QWidget, "Everblue", err.Error())
	}
	g.refreshVersions()
}

// installHash installs one version of Studio from the followed channel
func (g *gui) installHash(hash string) {
	if hash == "" {
		return
	}
	channel := g.settings.channel()
	g.runTask("Installing "+hash, "Installed "+hash, func(ctx context.Context, progress util.ProgressFunc, detach func()) error {
		return installVersion(ctx, g.home, g.logger, channel, hash, progress)
	})
}

// --- Fast flags ---

func (g *gui) buildFlags() *qt.QWidget {
	w, layout := page("Fast Flags", "Override Studio's feature flags. They're written into Studio each time it launches, so they survive updates and version switches.")

	g.flagSearch = qt.NewQLineEdit2()
	g.flagSearch.SetPlaceholderText("Search flags")
	g.flagSearch.SetClearButtonEnabled(true)
	g.flagSearch.OnTextChanged(func(string) { g.filterFlags() })
	add := button("Add Flag", "primary")
	add.OnClicked(g.addFlag)
	remove := button("Remove", "danger")
	remove.OnClicked(g.removeFlag)
	paste := button("Paste JSON", "")
	paste.SetToolTip("Add the flags from a ClientAppSettings.json copied to the clipboard")
	paste.OnClicked(func() { g.importFlags([]byte(qt.QGuiApplication_Clipboard().Text())) })
	importFile := button("Import", "")
	importFile.OnClicked(func() {
		path := qt.QFileDialog_GetOpenFileName4(g.manager.QWidget, "Import Fast Flags", g.home, "JSON (*.json);;All files (*)")
		if path == "" {
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			qt.QMessageBox_Critical(g.manager.QWidget, "Everblue", err.Error())
			return
		}
		g.importFlags(data)
	})
	export := button("Export", "")
	export.OnClicked(func() {
		path := qt.QFileDialog_GetSaveFileName4(g.manager.QWidget, "Export Fast Flags", g.home+"/ClientAppSettings.json", "JSON (*.json)")
		if path == "" {
			return
		}
		if err := os.WriteFile(path, fastFlagsJSON(g.tableFlags()), 0o644); err != nil {
			qt.QMessageBox_Critical(g.manager.QWidget, "Everblue", err.Error())
		}
	})
	toolbar := qt.NewQHBoxLayout2()
	toolbar.SetSpacing(10)
	toolbar.AddWidget2(g.flagSearch.QWidget, 1)
	for _, b := range []*qt.QPushButton{paste, importFile, export, remove, add} {
		toolbar.AddWidget(b.QWidget)
	}
	layout.AddLayout(toolbar.QLayout)

	g.flagTable = qt.NewQTableWidget3(0, 2)
	g.flagTable.SetHorizontalHeaderLabels([]string{"FLAG", "VALUE"})
	g.flagTable.HorizontalHeader().SetSectionResizeMode2(0, qt.QHeaderView__Stretch)
	g.flagTable.HorizontalHeader().SetSectionResizeMode2(1, qt.QHeaderView__Interactive)
	g.flagTable.HorizontalHeader().ResizeSection(1, 220)
	g.flagTable.HorizontalHeader().SetDefaultAlignment(qt.AlignLeft | qt.AlignVCenter)
	g.flagTable.VerticalHeader().Hide()
	g.flagTable.VerticalHeader().SetDefaultSectionSize(38)
	g.flagTable.SetShowGrid(false)
	g.flagTable.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	g.flagTable.OnItemChanged(func(*qt.QTableWidgetItem) {
		if !g.loadingFlag {
			g.saveFlags()
		}
	})
	layout.AddWidget2(g.flagTable.QWidget, 1)
	g.flagCount = label("", "faint")
	layout.AddWidget(g.flagCount.QWidget)
	g.loadFlags(loadFastFlags(g.home))
	return w
}

func (g *gui) loadFlags(flags []fastFlag) {
	g.loadingFlag = true
	g.flagTable.SetRowCount(0)
	for _, f := range flags {
		g.appendFlag(f.Name, flagText(f.Value))
	}
	g.loadingFlag = false
	g.filterFlags()
	g.updateFlagCount()
}

func (g *gui) appendFlag(name, value string) int {
	r := g.flagTable.RowCount()
	g.flagTable.InsertRow(r)
	g.flagTable.SetItem(r, 0, qt.NewQTableWidgetItem2(name))
	g.flagTable.SetItem(r, 1, qt.NewQTableWidgetItem2(value))
	return r
}

func (g *gui) tableFlags() []fastFlag {
	var flags []fastFlag
	for r := 0; r < g.flagTable.RowCount(); r++ {
		name, value := g.flagTable.Item(r, 0), g.flagTable.Item(r, 1)
		if name == nil || strings.TrimSpace(name.Text()) == "" {
			continue
		}
		text := ""
		if value != nil {
			text = value.Text()
		}
		flags = append(flags, fastFlag{strings.TrimSpace(name.Text()), flagValue(text)})
	}
	return flags
}

func (g *gui) saveFlags() {
	saveFastFlags(g.home, g.logger, g.tableFlags())
	g.updateFlagCount()
}

func (g *gui) updateFlagCount() {
	n := len(g.tableFlags())
	text := fmt.Sprintf("%d flags", n)
	if n == 1 {
		text = "1 flag"
	}
	g.flagCount.SetText(text + "  ·  saved to " + strings.Replace(fflagsPath(g.home), g.home, "~", 1))
}

func (g *gui) filterFlags() {
	query := strings.ToLower(strings.TrimSpace(g.flagSearch.Text()))
	for r := 0; r < g.flagTable.RowCount(); r++ {
		name := g.flagTable.Item(r, 0)
		hide := query != "" && (name == nil || !strings.Contains(strings.ToLower(name.Text()), query))
		g.flagTable.SetRowHidden(r, hide)
	}
}

func (g *gui) addFlag() {
	g.flagSearch.Clear()
	g.loadingFlag = true
	r := g.appendFlag("", "")
	g.loadingFlag = false
	g.flagTable.ScrollToBottom()
	g.flagTable.SetCurrentCell(r, 0)
	g.flagTable.EditItem(g.flagTable.Item(r, 0))
}

func (g *gui) removeFlag() {
	r := g.flagTable.CurrentRow()
	if r < 0 {
		return
	}
	g.flagTable.RemoveRow(r)
	g.saveFlags()
}

// importFlags merges a ClientAppSettings.json into the table, its values winning
func (g *gui) importFlags(data []byte) {
	incoming := parseFastFlags(data)
	if incoming == nil {
		qt.QMessageBox_Warning(g.manager.QWidget, "Everblue", "That isn't a JSON object of fast flags.")
		return
	}
	merged := map[string]any{}
	for _, f := range g.tableFlags() {
		merged[f.Name] = f.Value
	}
	for _, f := range incoming {
		merged[f.Name] = f.Value
	}
	flags := parseFastFlags(fastFlagsJSON(func() []fastFlag {
		var out []fastFlag
		for name, value := range merged {
			out = append(out, fastFlag{name, value})
		}
		return out
	}()))
	g.loadFlags(flags)
	g.saveFlags()
	g.setStatus(fmt.Sprintf("Imported %d flags", len(incoming)))
}

// --- Settings ---

func (g *gui) buildSettings() *qt.QWidget {
	w, layout := page("Settings", "")

	look, lookLayout := card("card")
	lookLayout.AddWidget(label("Appearance", "h2").QWidget)
	g.swatches = map[string]*qt.QPushButton{}
	swatchRow := qt.NewQHBoxLayout2()
	swatchRow.SetSpacing(14)
	for _, choice := range []struct{ key, name string }{{accentBlue, "Roblox Blue"}, {accentRed, "Roblox Red"}} {
		a := accentFor(choice.key)
		swatch := button("", "swatch")
		swatch.SetCheckable(true)
		swatch.SetStyleSheet(fmt.Sprintf("QPushButton#swatch { background: %s; border: 3px solid %s; } QPushButton#swatch:checked { border-color: #ffffff; }", a.base, colorCard))
		key := choice.key
		swatch.OnClicked(func() { g.setAccent(key) })
		g.swatches[key] = swatch
		swatchRow.AddWidget(swatch.QWidget)
		swatchRow.AddWidget(label(choice.name, "").QWidget)
		swatchRow.AddSpacing(14)
	}
	swatchRow.AddStretch()
	lookLayout.AddLayout(swatchRow.QLayout)
	g.swatches[g.settings.accentName()].SetChecked(true)
	animate := qt.NewQCheckBox3("Spin the logo")
	animate.SetChecked(!g.settings.StillLogo)
	animate.OnToggled(func(on bool) {
		g.settings.StillLogo = !on
		g.settings.save(g.home, g.logger)
		g.logo.setStill(!on)
		g.progress.logo.setStill(!on)
	})
	lookLayout.AddWidget(animate.QWidget)
	layout.AddWidget(look.QWidget)

	display, displayLayout := card("card")
	displayLayout.AddWidget(label("Display system", "h2").QWidget)
	options := []displaySystem{displayAsk, displayWayland, displayXWayland}
	group := displayOptions(displayLayout, options, g.settings.Display)
	group.OnIdClicked(func(id int) {
		g.settings.Display = options[id]
		g.settings.save(g.home, g.logger)
		g.refresh()
	})
	layout.AddWidget(display.QWidget)

	folders, foldersLayout := card("card")
	foldersLayout.AddWidget(label("Rosemary", "h2").QWidget)
	g.rosemaryValue = label("Checking…", "muted")
	prefix := studioPrefix(g.home).Dir
	openData := button("Everblue Folder", "")
	openData.OnClicked(g.openFolder(fmt.Sprintf("%s/.local/share/everblue", g.home)))
	openPrefix := button("Prefix", "")
	openPrefix.OnClicked(g.openFolder(prefix))
	foldersLayout.AddLayout(row(g.rosemaryValue.QWidget, nil, openData.QWidget, openPrefix.QWidget).QLayout)
	foldersLayout.AddWidget(label(strings.Replace(prefix, g.home, "~", 1), "mono").QWidget)
	layout.AddWidget(folders.QWidget)
	layout.AddStretch()
	return w
}

func (s settings) accentName() string {
	if _, ok := accents[s.Accent]; ok {
		return s.Accent
	}
	return accentBlue
}

func (g *gui) setAccent(key string) {
	g.settings.Accent = key
	g.settings.save(g.home, g.logger)
	for name, swatch := range g.swatches {
		swatch.SetChecked(name == key)
	}
	applyTheme(g.app, g.home, key)
	g.logo.setAccent(accentFor(key))
	g.progress.logo.setAccent(accentFor(key))
}

// --- Diagnostics ---

func (g *gui) buildDiagnostics() *qt.QWidget {
	w, layout := page("Diagnostics", "Checks for the things that most often stop Studio from installing or opening.")
	g.diagButton = button("Run Again", "primary")
	g.diagButton.OnClicked(g.diagnose)
	layout.AddLayout(row(g.diagButton.QWidget, nil).QLayout)
	g.diagResults = qt.NewQVBoxLayout2()
	g.diagResults.SetSpacing(12)
	layout.AddLayout(g.diagResults.QLayout)
	layout.AddStretch()
	return w
}

// diagnose runs the checks off the main thread, since finding libraries runs ldconfig
func (g *gui) diagnose() {
	g.diagButton.SetEnabled(false)
	g.setStatus("Running diagnostics…")
	go func() {
		checks := diagnose(context.Background(), g.home)
		for _, c := range checks {
			g.logger.Info("diagnostic", zap.String("check", c.Name), zap.Bool("ok", c.OK), zap.String("detail", c.Detail))
		}
		mainthread.Start(func() {
			g.diagButton.SetEnabled(true)
			g.showDiagnostics(checks)
			g.refresh()
		})
	}()
}

func (g *gui) showDiagnostics(checks []check) {
	for g.diagResults.Count() > 0 {
		item := g.diagResults.TakeAt(0)
		if w := item.Widget(); w != nil {
			w.DeleteLater()
		}
	}
	for _, c := range checks {
		frame, l := card("card")
		pill := label("OK", "pillGood")
		if !c.OK {
			pill = label("Needs attention", "pillWarn")
		}
		l.AddLayout(row(label(c.Name, "h2").QWidget, nil, pill.QWidget).QLayout)
		detail := label(c.Detail, "muted")
		detail.SetWordWrap(true)
		detail.SetTextInteractionFlags(qt.TextSelectableByMouse)
		l.AddWidget(detail.QWidget)
		if c.Fix != "" && !c.OK {
			text, action := "Install Base System", g.installBase
			if c.Fix == fixRosemary {
				text, action = "Install Studio Rosemary", g.installRosemary
			}
			fix := button(text, "primary")
			fix.OnClicked(action)
			fix.SetEnabled(!g.busy)
			l.AddLayout(row(fix.QWidget, nil).QLayout)
		}
		g.diagResults.AddWidget(frame.QWidget)
	}
}

func (g *gui) installRosemary() {
	g.runTask("Installing rosemary", "Installed rosemary", func(ctx context.Context, progress util.ProgressFunc, detach func()) error {
		if err := installRosemary(ctx, g.logger, progress); err != nil {
			return err
		}
		mainthread.Start(g.diagnose)
		return nil
	})
}

func (g *gui) installBase() {
	g.runTask("Installing base system", "Installed base system", func(ctx context.Context, progress util.ProgressFunc, detach func()) error {
		progress("Running rosemary prefix create", 0, -1)
		return installBase(ctx, g.home, g.logger)
	})
}

// --- Logs ---

func (g *gui) buildLogs() *qt.QWidget {
	w, layout := page("Logs", "")
	setName(g.logPane.QObject, "log")
	g.logPane.SetReadOnly(true)
	g.logPane.SetMaximumBlockCount(5000)
	g.logPane.SetLineWrapMode(qt.QPlainTextEdit__NoWrap)
	layout.AddWidget2(g.logPane.QWidget, 1)
	return w
}

// --- State ---

// checkForUpdates looks up the latest Studio on the followed channel and rosemary's version off the main thread
func (g *gui) checkForUpdates() {
	channel := g.settings.channel()
	g.latestValue.SetText("Checking…")
	g.channelLatest.SetText("Checking " + channel + "…")
	g.checkAction.SetEnabled(false)
	g.setStatus("Checking for updates…")
	go func() {
		ctx := context.Background()
		rosemary, rosemaryErr := util.RosemaryVersion(ctx)
		_, studioBuild := util.RosemaryState(ctx)
		if rosemaryErr != nil {
			g.logger.Error("failed to get rosemary version", zap.Error(rosemaryErr))
		}
		data, err := util.GetChannelVersion(ctx, g.logger, channel)
		mainthread.Start(func() {
			g.checkAction.SetEnabled(true)
			if rosemaryErr != nil {
				g.rosemaryValue.SetText("rosemary isn't on PATH")
			} else {
				// the first line is the version, the rest is build details
				first, _, _ := strings.Cut(strings.TrimSpace(rosemary), "\n")
				if studioBuild {
					first += "  ·  Studio build"
				} else {
					first += "  ·  upstream build, Everblue installs the Studio build at launch"
				}
				g.rosemaryValue.SetText(first)
			}
			g.latest = ""
			switch {
			case errors.Is(err, util.ErrPrivateChannel):
				g.latestValue.SetText("Private channel")
				g.channelLatest.SetText(channel + " is private to Roblox")
			case err != nil:
				g.latestValue.SetText("Unavailable")
				g.channelLatest.SetText("Couldn't reach Roblox")
			default:
				g.latest = data.ClientVersionUpload
				g.latestValue.SetText(data.Version)
				g.channelLatest.SetText(fmt.Sprintf("Latest on %s: %s (%s)", channel, data.Version, data.ClientVersionUpload))
			}
			g.refresh()
		})
	}()
}

func (g *gui) refresh() {
	installed := installedVersion(g.home)
	hasStudio := studioInstalled(g.home) && installed != ""

	install, status, pill := "Update", "Update available", "pillAccent"
	switch {
	case !hasStudio:
		install, status, pill = "Install Studio", "Not installed", "pillWarn"
	case g.latest == "":
		install, status, pill = "Reinstall", "Installed", "pillGood"
	case installed == g.latest:
		install, status, pill = "Reinstall", "Up to date", "pillGood"
	}
	g.statusPill.SetText(status)
	setName(g.statusPill.QObject, pill)
	g.statusPill.Style().Unpolish(g.statusPill.QWidget)
	g.statusPill.Style().Polish(g.statusPill.QWidget)
	g.installButton.SetText(install)

	if hasStudio {
		version := appVersion(studioApp(g.home))
		if version == "" {
			version = shortHash(installed)
		}
		g.heroTitle.SetText("Studio " + version)
		g.heroDetail.SetText(fmt.Sprintf("%s  ·  %s channel", installed, g.settings.channel()))
	} else {
		g.heroTitle.SetText("Get Roblox Studio")
		g.heroDetail.SetText("Everblue installs Studio into a rosemary prefix and keeps it up to date.")
	}
	g.channelValue.SetText(g.settings.channel())
	g.displayValue.SetText(g.settings.Display.String())

	// installing creates the prefix, so this can change along with the installed version
	pfx := studioPrefix(g.home)
	if err := pfx.ResolveYAML(); err != nil {
		g.baseValue.SetText("Not created yet")
	} else if pfx.State.Base.Project == "" {
		g.baseValue.SetText("Missing (see Diagnostics)")
	} else {
		g.baseValue.SetText(fmt.Sprintf("%s %s", pfx.State.Base.Project, pfx.State.Base.Tag))
	}

	g.runAction.SetEnabled(!g.busy && hasStudio)
	g.playButton.SetEnabled(!g.busy && hasStudio)
	g.installButton.SetEnabled(!g.busy)
	g.refreshVersions()
	if !g.busy {
		g.setStatus("Ready")
	}
}

// install puts the newest Studio of the followed channel in place. A first install from LIVE goes through
// Roblox's installer; anything else downloads the whole app, since the installer only knows LIVE.
func (g *gui) install() {
	if g.busy {
		return
	}
	channel := g.settings.channel()
	if (studioInstalled(g.home) || !strings.EqualFold(channel, util.LiveChannel)) && g.latest != "" {
		g.installHash(g.latest)
		return
	}
	g.runTask("Installing Studio", "Installer is running", func(ctx context.Context, progress util.ProgressFunc, detach func()) error {
		return installStudio(ctx, g.home, g.logger, g.installDisplay(), func(stage string, done, total int64) {
			// the installer opens its own window, don't sit on top of it
			if stage == "Running installer" {
				detach()
			}
			progress(stage, done, total)
		})
	})
}

// launch asks which display system to use unless one is saved, then runs Studio. It reports whether Studio was started.
func (g *gui) launch() bool {
	display := g.settings.Display
	if display == displayAsk {
		var ok bool
		if display, ok = g.chooseDisplay(); !ok {
			return false
		}
	}
	g.runTask("Launching Studio", "Studio is running", func(ctx context.Context, progress util.ProgressFunc, detach func()) error {
		progress(fmt.Sprintf("%s, %s", installedVersion(g.home), display), 0, -1)
		detach()
		return runStudio(ctx, g.home, g.logger, display)
	})
	return true
}

// installDisplay is the display system for the Studio the installer launches. Asking here as well as when
// Studio is next run would ask twice for one install, so without a saved choice it's Wayland, rosemary's own default.
func (g *gui) installDisplay() displaySystem {
	if g.settings.Display == displayAsk {
		return displayWayland
	}
	return g.settings.Display
}

// displayOptions builds a radio button with an explanation under it for each display system
func displayOptions(layout *qt.QVBoxLayout, options []displaySystem, selected displaySystem) *qt.QButtonGroup {
	describe := map[displaySystem]string{
		displayAsk: "Choose before Studio opens each time.",
		displayWayland: "Native Wayland windows. Wayland doesn't tell programs where their windows are, " +
			"so dragging a floating panel back into Studio can't work on most desktops.",
		displayXWayland: "Runs Studio as an X11 program. Window positions are known on every desktop, " +
			"so panels can be docked by dragging, but X11 programs can look and behave worse on Wayland desktops.",
	}
	if desktop := compositorDesktop(); desktop != "" {
		describe[displayWayland] = "Native Wayland windows. On " + desktop + ", rosemary asks the compositor where " +
			"windows are so floating panels can be dragged back into Studio."
	}
	group := qt.NewQButtonGroup()
	for i, option := range options {
		radio := qt.NewQRadioButton3(option.String())
		radio.SetChecked(option == selected)
		group.AddButton2(radio.QAbstractButton, i)
		detail := label(describe[option], "faint")
		detail.SetWordWrap(true)
		// line the explanation up with the radio button's text
		detail.SetIndent(30)
		layout.AddWidget(radio.QWidget)
		layout.AddWidget(detail.QWidget)
		layout.AddSpacing(6)
	}
	return group
}

// chooseDisplay asks which display system to launch Studio with, and saves it when asked to remember
func (g *gui) chooseDisplay() (displaySystem, bool) {
	d := qt.NewQDialog(g.manager.QWidget)
	d.SetWindowTitle("Launch Studio")
	d.SetMinimumWidth(500)
	layout := qt.NewQVBoxLayout2()
	layout.SetContentsMargins(26, 24, 26, 20)
	layout.SetSpacing(8)
	layout.AddWidget(label("How should Studio's windows be shown?", "h2").QWidget)
	layout.AddSpacing(8)
	options := []displaySystem{displayWayland, displayXWayland}
	group := displayOptions(layout, options, displayWayland)
	remember := qt.NewQCheckBox3("Remember my choice")
	remember.SetToolTip("Change it later in Everblue's Settings.")
	layout.AddWidget(remember.QWidget)
	layout.AddSpacing(8)
	cancel := button("Cancel", "")
	cancel.OnClicked(d.Reject)
	launch := button("▶  Launch", "primary")
	launch.OnClicked(d.Accept)
	launch.SetDefault(true)
	layout.AddLayout(row(nil, cancel.QWidget, launch.QWidget).QLayout)
	d.SetLayout(layout.QLayout)
	if d.Exec() != int(qt.QDialog__Accepted) {
		return displayAsk, false
	}
	choice := options[group.CheckedId()]
	if remember.IsChecked() {
		g.settings.Display = choice
		g.settings.save(g.home, g.logger)
	}
	return choice, true
}

// runTask shows the progress dialog while fn runs in a goroutine. Once fn calls detach and keeps
// running for detachDelay, the dialog hides and the sidebar shows running instead.
func (g *gui) runTask(title, running string, fn func(ctx context.Context, progress util.ProgressFunc, detach func()) error) {
	ctx, cancel := context.WithCancel(context.Background())
	d := g.progress
	d.cancel = cancel
	d.status.SetText(title)
	d.info.SetText("")
	d.bar.SetRange(0, 0)
	g.busy = true
	g.refresh()
	g.setStatus(title + "…")
	d.Show()

	lastStage, lastPercent := "", int64(-2)
	progress := func(stage string, done, total int64) {
		percent := int64(-1)
		if total > 0 {
			percent = done * 100 / total
		}
		// downloads call this per chunk, only bother Qt when something visibly changes
		if stage == lastStage && percent == lastPercent {
			return
		}
		lastStage, lastPercent = stage, percent
		mainthread.Start(func() {
			text := stage
			if percent >= 0 && total > 1<<20 {
				text = fmt.Sprintf("%s  ·  %d / %d MB", stage, done>>20, total>>20)
			}
			d.info.SetText(text)
			if percent < 0 {
				d.bar.SetRange(0, 0)
				return
			}
			d.bar.SetRange(0, 100)
			d.bar.SetValue(int(percent))
		})
	}

	detach := func() {
		go func() {
			select {
			case <-time.After(detachDelay):
				mainthread.Start(func() {
					if ctx.Err() == nil {
						d.Hide()
						g.setStatus(running)
					}
				})
			case <-ctx.Done():
			}
		}()
	}

	go func() {
		err := fn(ctx, progress, detach)
		mainthread.Start(func() {
			cancelled := ctx.Err() != nil
			cancel()
			d.Hide()
			g.busy = false
			g.refresh()
			if err != nil && !cancelled {
				g.logger.Error(title+" failed", zap.Error(err))
				qt.QMessageBox_Critical(g.manager.QWidget, "Everblue", fmt.Sprintf("%s failed: %v", title, err))
			}
			if g.quitWhenDone {
				qt.QCoreApplication_Quit()
			}
		})
	}()
}

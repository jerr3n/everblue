package main

import (
	"fmt"
	"os"
	"strings"

	qt "github.com/mappu/miqt/qt6"
)

// The compact layout is one narrow column of grouped rows under a header bar, the way GNOME lays out
// preference windows. Pages other than the main one open in its place, with a back button in the header.

const compactMaxWidth = 600

var pageTitles = [...]string{pageHome: "Everblue", pageVersions: "Versions", pageFlags: "Fast Flags",
	pageSettings: "Settings", pageDiagnostics: "Diagnostics", pageLogs: "Logs"}

// iconPaths are stroked 24×24 glyphs for the compact layout's buttons
var iconPaths = map[string]string{
	"back":     `<path d="M15 5l-7 7 7 7"/>`,
	"next":     `<path d="M5 12h14M13 6l6 6-6 6"/>`,
	"close":    `<path d="M7 7l10 10M17 7 7 17"/>`,
	"menu":     `<circle cx="12" cy="5.5" r="1.6" fill="@"/><circle cx="12" cy="12" r="1.6" fill="@"/><circle cx="12" cy="18.5" r="1.6" fill="@"/>`,
	"folder":   `<path d="M3.5 7.5a2 2 0 0 1 2-2h4l2 2h7a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-13a2 2 0 0 1-2-2z"/>`,
	"download": `<path d="M12 4v11M7 10l5 5 5-5M5 19.5h14"/>`,
}

// iconColor is the text color of the current theme, which is the desktop's when following it
func iconColor() string {
	return qt.QGuiApplication_Palette().ColorWithCr(qt.QPalette__ButtonText).Name()
}

func icon(name string, size int) *qt.QIcon {
	color := iconColor()
	body := iconPaths[name]
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="%s" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">%s</svg>`,
		color, body)
	svg = strings.ReplaceAll(svg, `fill="@"`, fmt.Sprintf(`fill="%s"`, color))
	return qt.NewQIcon2(svgPixmap(svg, size))
}

// iconButton is a square button showing one glyph
func iconButton(name, style, tip string) *qt.QPushButton {
	b := button("", style)
	b.SetIcon(icon(name, 18))
	b.SetIconSize(qt.NewQSize2(18, 18))
	b.SetToolTip(tip)
	return b
}

// boxedList is a rounded panel of rows with lines between them
type boxedList struct {
	frame  *qt.QFrame
	layout *qt.QVBoxLayout
	links  []*qt.QWidget
	rows   []*qt.QWidget
}

// group adds a titled list to a page, with any extra widgets at the right of its title
func group(page *qt.QVBoxLayout, title string, extra ...*qt.QWidget) *boxedList {
	page.AddSpacing(14)
	header := qt.NewQHBoxLayout2()
	header.SetContentsMargins(4, 0, 4, 0)
	header.AddWidget(label(title, "groupTitle").QWidget)
	header.AddStretch()
	for _, w := range extra {
		header.AddWidget(w)
	}
	page.AddLayout(header.QLayout)
	page.AddSpacing(4)
	b := &boxedList{frame: qt.NewQFrame2(), layout: qt.NewQVBoxLayout2()}
	setName(b.frame.QObject, "boxed")
	b.layout.SetContentsMargins(0, 0, 0, 0)
	b.layout.SetSpacing(0)
	b.frame.SetLayout(b.layout.QLayout)
	page.AddWidget(b.frame.QWidget)
	return b
}

func (b *boxedList) add(w *qt.QWidget) {
	if len(b.rows) > 0 {
		sep := qt.NewQFrame2()
		setName(sep.QObject, "rowSeparator")
		b.layout.AddWidget(sep.QWidget)
	}
	b.rows = append(b.rows, w)
	b.layout.AddWidget(w)
}

// addLink adds a row that does something when clicked, such as opening a page
func (b *boxedList) addLink(w *qt.QWidget, fn func()) {
	w.SetAttribute(qt.WA_Hover)
	w.SetAttribute(qt.WA_StyledBackground)
	w.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	w.OnMouseReleaseEvent(func(super func(*qt.QMouseEvent), e *qt.QMouseEvent) {
		super(e)
		if e.Button() == qt.LeftButton && w.Rect().ContainsWithQPoint(e.Pos()) {
			fn()
		}
	})
	b.links = append(b.links, w)
	b.add(w)
}

// round names the links at the ends of the list so their hover follows its rounded corners
func (b *boxedList) round() {
	last := len(b.rows) - 1
	for _, w := range b.links {
		name := "rowLink"
		switch {
		case last == 0:
			name = "rowLinkOnly"
		case w == b.rows[0]:
			name = "rowLinkTop"
		case w == b.rows[last]:
			name = "rowLinkBottom"
		}
		setName(w.QObject, name)
	}
}

// prefRow is a title over a subtitle with widgets at the right. The subtitle hides while it's empty.
func prefRow(title, subtitle string, suffix ...*qt.QWidget) (row *qt.QWidget, titleLabel, subtitleLabel *qt.QLabel) {
	row = qt.NewQWidget2()
	setName(row.QObject, "row")
	row.SetMinimumHeight(56)
	layout := qt.NewQHBoxLayout2()
	layout.SetContentsMargins(14, 9, 12, 9)
	layout.SetSpacing(10)
	text := qt.NewQVBoxLayout2()
	text.SetSpacing(2)
	titleLabel = label(title, "rowTitle")
	titleLabel.SetWordWrap(true)
	subtitleLabel = label(subtitle, "rowSubtitle")
	subtitleLabel.SetWordWrap(true)
	text.AddStretch()
	text.AddWidget(titleLabel.QWidget)
	text.AddWidget(subtitleLabel.QWidget)
	text.AddStretch()
	layout.AddLayout2(text.QLayout, 1)
	for _, w := range suffix {
		layout.AddWidget3(w, 0, qt.AlignVCenter)
	}
	row.SetLayout(layout.QLayout)
	return row, titleLabel, subtitleLabel
}

// linkRow is a row for a page, with an arrow at its right
func linkRow(title, subtitle string, suffix ...*qt.QWidget) (*qt.QWidget, *qt.QLabel) {
	arrow := qt.NewQLabel2()
	arrow.SetPixmap(icon("next", 18).Pixmap2(18, 18))
	row, _, sub := prefRow(title, subtitle, append(suffix, arrow.QWidget)...)
	return row, sub
}

// switchBox is a toggle for a row's right
func switchBox(on bool, toggled func(bool)) *qt.QWidget {
	s := qt.NewQCheckBox2()
	setName(s.QObject, "switch")
	s.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	s.SetChecked(on)
	s.OnToggled(toggled)
	return s.QWidget
}

// dropDown is a borderless choice for a row's right
func dropDown(items []string, current int, chosen func(int)) *qt.QComboBox {
	c := qt.NewQComboBox2()
	setName(c.QObject, "flat")
	c.AddItems(items)
	c.SetCurrentIndex(current)
	c.SetSizeAdjustPolicy(qt.QComboBox__AdjustToContents)
	c.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	c.OnActivated(chosen)
	return c
}

// centered keeps a page from growing past a readable width in a wide window
func centered(w *qt.QWidget, width int) *qt.QWidget {
	w.SetMaximumWidth(width)
	outer := qt.NewQWidget2()
	setName(outer.QObject, "page")
	layout := qt.NewQHBoxLayout2()
	layout.SetContentsMargins(0, 0, 0, 0)
	layout.AddStretch()
	layout.AddWidget2(w, 100)
	layout.AddStretch()
	outer.SetLayout(layout.QLayout)
	return outer
}

// subpage drops a page's own title, since the header bar shows it
func subpage(w *qt.QWidget) *qt.QWidget {
	w.Layout().ItemAt(0).Widget().Hide()
	w.Layout().SetContentsMargins(20, 8, 20, 20)
	return centered(w, compactMaxWidth)
}

func (g *gui) buildCompact() {
	g.manager.Resize(540, 720)
	g.manager.SetMinimumSize2(420, 480)

	g.pages = qt.NewQStackedWidget2()
	g.pages.AddWidget(scrolled(centered(g.buildCompactMain(), compactMaxWidth)))
	g.pages.AddWidget(scrolled(subpage(g.buildVersions())))
	g.pages.AddWidget(subpage(g.buildFlags()))
	// settings are on the main page
	g.pages.AddWidget(qt.NewQWidget2())
	g.pages.AddWidget(scrolled(subpage(g.buildDiagnostics())))
	g.pages.AddWidget(subpage(g.buildLogs()))

	body := qt.NewQVBoxLayout2()
	body.SetContentsMargins(0, 0, 0, 0)
	body.SetSpacing(0)
	body.AddWidget(g.buildHeaderBar())
	body.AddWidget2(g.pages.QWidget, 1)
	central := qt.NewQWidget2()
	central.SetLayout(body.QLayout)
	g.manager.SetCentralWidget(central)
}

// buildHeaderBar is the compact layout's title row: back when on a page, the title, a menu and close
func (g *gui) buildHeaderBar() *qt.QWidget {
	bar := qt.NewQWidget2()
	setName(bar.QObject, "headerBar")
	bar.SetAttribute(qt.WA_StyledBackground)
	layout := qt.NewQGridLayout2()
	layout.SetContentsMargins(10, 8, 10, 6)

	g.backButton = iconButton("back", "flatIcon", "Back (Alt+Left)")
	g.backButton.OnClicked(func() { g.showPage(pageHome) })
	left := qt.NewQHBoxLayout2()
	left.AddWidget(g.backButton.QWidget)
	left.AddStretch()

	g.headerTitle = label(pageTitles[pageHome], "headerTitle")
	g.headerTitle.SetAlignment(qt.AlignCenter)

	menu := qt.NewQMenu(bar)
	menu.AddActionWithText("Check for Updates").OnTriggered(g.checkForUpdates)
	menu.AddActionWithText("Open Everblue Folder").OnTriggered(g.openFolder(fmt.Sprintf("%s/.local/share/everblue", g.home)))
	menu.AddSeparator()
	menu.AddActionWithText("Diagnostics").OnTriggered(func() { g.showPage(pageDiagnostics) })
	menu.AddActionWithText("Logs").OnTriggered(func() { g.showPage(pageLogs) })
	menuButton := iconButton("menu", "flatIcon", "Menu")
	menuButton.SetMenu(menu)
	closeButton := iconButton("close", "closeIcon", "Close")
	closeButton.SetIconSize(qt.NewQSize2(14, 14))
	closeButton.OnClicked(func() { g.manager.Close() })
	right := qt.NewQHBoxLayout2()
	right.SetSpacing(6)
	right.AddStretch()
	right.AddWidget(menuButton.QWidget)
	right.AddWidget(closeButton.QWidget)

	layout.AddLayout(left.QLayout, 0, 0)
	layout.AddWidget2(g.headerTitle.QWidget, 0, 1)
	layout.AddLayout(right.QLayout, 0, 2)
	// equal sides keep the title centred whatever is on either side of it
	layout.SetColumnStretch(0, 1)
	layout.SetColumnStretch(2, 1)
	bar.SetLayout(layout.QLayout)
	return bar
}

func (g *gui) buildCompactMain() *qt.QWidget {
	w := qt.NewQWidget2()
	setName(w.QObject, "page")
	layout := qt.NewQVBoxLayout2()
	layout.SetContentsMargins(20, 0, 20, 20)
	layout.SetSpacing(0)

	g.statusPill = label("", "pillGood")
	studio := group(layout, "Roblox Studio", g.statusPill.QWidget)
	g.installButton = button("Install", "pillPlain")
	g.installButton.OnClicked(g.install)
	g.playButton = button("▶  Play", "pill")
	g.playButton.OnClicked(func() { g.launch() })
	hero, title, detail := prefRow("", "", g.installButton.QWidget, g.playButton.QWidget)
	g.heroTitle, g.heroDetail = title, detail
	g.heroTitle.SetWordWrap(true)
	hero.SetMinimumHeight(68)
	studio.add(hero)
	g.channelValue = label("", "rowValue")
	latest, latestValue := linkRow("Latest Version", "Checking…", g.channelValue.QWidget)
	g.latestValue = latestValue
	studio.addLink(latest, func() { g.showPage(pageVersions) })
	flags, _ := linkRow("Fast Flags", "Override Studio's feature flags")
	studio.addLink(flags, func() { g.showPage(pageFlags) })
	studio.round()

	display := group(layout, "Display")
	options := []displaySystem{displayAsk, displayWayland, displayXWayland}
	names := make([]string, len(options))
	current := 0
	for i, o := range options {
		names[i] = o.String()
		if o == g.settings.Display {
			current = i
		}
	}
	g.displayBox = dropDown(names, current, func(i int) {
		g.settings.Display = options[i]
		g.settings.save(g.home, g.logger)
		g.refresh()
	})
	displayRow, _, _ := prefRow("Display System", "How Studio's windows are shown", g.displayBox.QWidget)
	display.add(displayRow)

	look := group(layout, "Appearance")
	accentKeys := []string{accentBlue, accentRed}
	currentAccent := 0
	if g.settings.accentName() == accentRed {
		currentAccent = 1
	}
	accentBox := dropDown([]string{"Roblox Blue", "Roblox Red"}, currentAccent, func(i int) { g.setAccent(accentKeys[i]) })
	accentRow, _, _ := prefRow("Accent Color", "Used for buttons, switches and the logo", accentBox.QWidget)
	if g.settings.Theme != themeEverblue {
		accentRow.SetEnabled(false)
		accentRow.SetToolTip("The theme decides the accent color")
	}
	themeBox := dropDown(themeNames, themeIndex(g.settings.Theme), func(i int) { g.setTheme(themeKeys[i]) })
	themeRow, _, _ := prefRow("Theme", "System Colors keeps this look in your desktop's colors", themeBox.QWidget)
	look.add(themeRow)
	look.add(accentRow)
	spin, _, _ := prefRow("Spin the Logo", "Animate the logo while Studio installs or launches", switchBox(!g.settings.StillLogo, func(on bool) {
		g.settings.StillLogo = !on
		g.settings.save(g.home, g.logger)
		g.progress.logo.setStill(!on)
	}))
	look.add(spin)
	classic, _, _ := prefRow("Classic Layout", "Use the wider window with a sidebar", switchBox(false, func(on bool) { g.setClassic(on) }))
	look.add(classic)

	openData := iconButton("folder", "flatIcon", "Open Everblue's folder")
	openData.OnClicked(g.openFolder(fmt.Sprintf("%s/.local/share/everblue", g.home)))
	rosemary := group(layout, "Rosemary", openData.QWidget)
	openPrefix := iconButton("folder", "iconButton", "Open the prefix")
	openPrefix.OnClicked(func() {
		prefix := studioPrefix(g.home).Dir
		os.MkdirAll(prefix, 0o755)
		g.openFolder(prefix)()
	})
	rosemaryRow, _, rosemaryValue := prefRow("Rosemary", "Checking…", openPrefix.QWidget)
	g.rosemaryValue = rosemaryValue
	rosemary.add(rosemaryRow)
	baseRow, _, baseValue := prefRow("Base System", "")
	g.baseValue = baseValue
	rosemary.add(baseRow)
	diag, _ := linkRow("Diagnostics", "Check for what stops Studio installing or opening")
	rosemary.addLink(diag, func() { g.showPage(pageDiagnostics) })
	logs, _ := linkRow("Logs", "What Everblue has done this session")
	rosemary.addLink(logs, func() { g.showPage(pageLogs) })
	rosemary.round()

	layout.AddSpacing(14)
	g.footer = label("Ready", "footer")
	g.footer.SetAlignment(qt.AlignCenter)
	g.footer.SetWordWrap(true)
	layout.AddWidget(g.footer.QWidget)
	layout.AddStretch()
	w.SetLayout(layout.QLayout)
	return w
}

package main

import (
	"fmt"
	"os"
	"strings"

	qt "github.com/mappu/miqt/qt6"
)

// Everblue draws its own dark look rather than following the desktop's Qt theme: Fusion underneath so
// every widget can be restyled, a dark palette for what the stylesheet doesn't reach, and one accent.
// Each layout has its own grays: navy blacks for the classic sidebar window, and the neutral grays of
// GNOME's dark style for the compact one.

const (
	accentBlue = "blue"
	accentRed  = "red"
)

// accent is a highlight color and its lighter and darker shades
type accent struct {
	base, hover, pressed string
	r, g, b              int
}

var accents = map[string]accent{
	accentBlue: {"#335fff", "#4f75ff", "#2a4fd6", 0x33, 0x5f, 0xff},
	accentRed:  {"#e2231a", "#f0443b", "#c01c14", 0xe2, 0x23, 0x1a},
}

func accentFor(name string) accent {
	if a, ok := accents[name]; ok {
		return a
	}
	return accents[accentBlue]
}

// neutrals are a layout's grays and text colors
type neutrals struct {
	window, sidebar, card, raised, input, border, hover, text, muted, faint string
}

var classicColors = neutrals{
	window: "#0d0f13", sidebar: "#12151b", card: "#161a21", raised: "#1d222b", input: "#0f1217",
	border: "#252b36", hover: "#2b3240", text: "#e8eaef", muted: "#8b93a3", faint: "#5d6574",
}

var compactColors = neutrals{
	window: "#222226", sidebar: "#2e2e32", card: "#333337", raised: "#47474b", input: "#2a2a2e",
	border: "#3a3a3e", hover: "#55555a", text: "#ffffff", muted: "#a5a5a8", faint: "#7c7c80",
}

// desktop holds the desktop's Qt colors, taken before Everblue replaces the palette with its own
var desktop struct {
	taken                                   bool
	window, base, text, highlight, highText string
}

func takeDesktopColors() {
	if desktop.taken {
		return
	}
	p := qt.QGuiApplication_Palette()
	name := func(role qt.QPalette__ColorRole) string { return p.ColorWithCr(role).Name() }
	desktop.window, desktop.base, desktop.text = name(qt.QPalette__Window), name(qt.QPalette__Base), name(qt.QPalette__WindowText)
	desktop.highlight, desktop.highText = name(qt.QPalette__Highlight), name(qt.QPalette__HighlightedText)
	desktop.taken = true
}

// mix blends two #rrggbb colors, t of the way from a to b
func mix(a, b string, t float64) string {
	var ar, ag, ab, br, bg, bb int
	fmt.Sscanf(a, "#%02x%02x%02x", &ar, &ag, &ab)
	fmt.Sscanf(b, "#%02x%02x%02x", &br, &bg, &bb)
	blend := func(x, y int) int { return int(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return fmt.Sprintf("#%02x%02x%02x", blend(ar, br), blend(ag, bg), blend(ab, bb))
}

// desktopNeutrals builds a layout's grays from the desktop's window and text colors, so the panels sit
// a little toward the text color whether the desktop is light or dark
func desktopNeutrals() neutrals {
	w, t := desktop.window, desktop.text
	return neutrals{
		window: w, sidebar: mix(w, t, 0.04), card: mix(w, t, 0.08), raised: mix(w, t, 0.16), input: desktop.base,
		border: mix(w, t, 0.13), hover: mix(w, t, 0.24), text: t, muted: mix(t, w, 0.35), faint: mix(t, w, 0.55),
	}
}

// themeAccent is the chosen accent, or the desktop's highlight color when following its colors
func themeAccent(s settings) accent {
	if s.Theme != themeColors {
		return accentFor(s.Accent)
	}
	h := desktop.highlight
	var r, g, b int
	fmt.Sscanf(h, "#%02x%02x%02x", &r, &g, &b)
	return accent{h, mix(h, "#ffffff", 0.15), mix(h, "#000000", 0.15), r, g, b}
}

func colorsFor(s settings) neutrals {
	if s.Theme == themeColors {
		return desktopNeutrals()
	}
	if s.Classic {
		return classicColors
	}
	return compactColors
}

const (
	colorGood = "#3ccf8e"
	colorWarn = "#f5b544"
)

func (a accent) rgba(alpha float64) string {
	return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", a.r, a.g, a.b, alpha)
}

// themeImages writes the few glyphs the stylesheet needs as files, since a stylesheet can only name images.
// The names carry their colors, because Qt caches an image by its path.
func themeImages(home string, n neutrals, a accent) map[string]string {
	dir := fmt.Sprintf("%s/.cache/everblue", home)
	os.MkdirAll(dir, 0o755)
	hex := func(color string) string { return strings.TrimPrefix(color, "#") }
	images := map[string]string{
		"@arrow":     fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 12 12"><path d="M2.5 4.5 6 8l3.5-3.5" fill="none" stroke="%s" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>`, n.muted),
		"@check":     `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 18 18"><path d="M4.5 9.5 7.5 12.5 13.5 5.5" fill="none" stroke="#ffffff" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
		"@switchOn":  fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 46 26"><rect width="46" height="26" rx="13" fill="%s"/><circle cx="33" cy="13" r="10" fill="#ffffff"/></svg>`, a.base),
		"@switchOff": fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 46 26"><rect width="46" height="26" rx="13" fill="%s"/><circle cx="13" cy="13" r="10" fill="#d2d2d3"/></svg>`, n.hover),
	}
	names := map[string]string{
		"@arrow": "arrow-" + hex(n.muted), "@check": "check",
		"@switchOn": "switch-on-" + hex(a.base), "@switchOff": "switch-off-" + hex(n.hover),
	}
	paths := map[string]string{}
	for key, svg := range images {
		path := fmt.Sprintf("%s/%s.svg", dir, names[key])
		os.WriteFile(path, []byte(svg), 0o644)
		paths[key] = path
	}
	return paths
}

func styleSheet(n neutrals, a accent, images map[string]string) string {
	return strings.NewReplacer(
		"@arrow", images["@arrow"], "@check", images["@check"],
		"@switchOn", images["@switchOn"], "@switchOff", images["@switchOff"],
		"@window", n.window, "@sidebar", n.sidebar, "@card", n.card, "@raised", n.raised,
		"@input", n.input, "@border", n.border, "@hover", n.hover, "@text", n.text,
		"@muted", n.muted, "@faint", n.faint, "@good", colorGood, "@warn", colorWarn,
		"@accentSoft", a.rgba(0.16), "@accentLine", a.rgba(0.55), "@accentHover", a.hover,
		"@accentPressed", a.pressed, "@accent", a.base,
	).Replace(`
QMainWindow, QDialog, QMessageBox { background: @window; }
QWidget { color: @text; font-size: 10pt; selection-background-color: @accent; selection-color: #ffffff; }
QLabel { background: transparent; }
QToolTip { background: @raised; color: @text; border: 1px solid @border; border-radius: 6px; padding: 6px 8px; }

#sidebar { background: @sidebar; border-right: 1px solid @border; }
#brand { font-size: 15pt; font-weight: 700; }
#brandSub { color: @faint; font-size: 8pt; letter-spacing: 1px; }
QPushButton#nav {
	text-align: left; padding: 10px 14px; border: none; border-radius: 9px;
	color: @muted; background: transparent; font-size: 10.5pt; font-weight: 500;
}
QPushButton#nav:hover { background: @raised; color: @text; }
QPushButton#nav:checked { background: @accentSoft; color: #ffffff; font-weight: 600; }
#sidebarFooter { color: @faint; font-size: 8.5pt; }

#page { background: transparent; }
#h1 { font-size: 24pt; font-weight: 800; }
#h2 { font-size: 13pt; font-weight: 700; }
#muted { color: @muted; }
#faint { color: @faint; font-size: 9pt; }
#caption { color: @faint; font-size: 8pt; font-weight: 700; letter-spacing: 1px; }
#value { font-size: 12pt; font-weight: 600; }
#mono { font-family: monospace; color: @muted; }

QFrame#card { background: @card; border: 1px solid @border; border-radius: 14px; }
QFrame#hero {
	border: 1px solid @border; border-radius: 18px;
	background: qlineargradient(x1:0, y1:0, x2:1, y2:1, stop:0 @accentSoft, stop:0.55 @card, stop:1 @card);
}

#pillGood { color: @good; background: rgba(60, 207, 142, 0.12); border-radius: 10px; padding: 3px 10px; font-weight: 600; }
#pillWarn { color: @warn; background: rgba(245, 181, 68, 0.12); border-radius: 10px; padding: 3px 10px; font-weight: 600; }
#pillAccent { color: #ffffff; background: @accentSoft; border-radius: 10px; padding: 3px 10px; font-weight: 600; }

QPushButton {
	background: @raised; border: 1px solid @border; border-radius: 9px;
	padding: 8px 16px; color: @text; font-weight: 500;
}
QPushButton:hover { background: @hover; }
QPushButton:pressed { background: @border; }
QPushButton:disabled { color: @faint; background: @card; border-color: @card; }
QPushButton#primary { background: @accent; border: 1px solid @accent; color: #ffffff; font-weight: 600; }
QPushButton#primary:hover { background: @accentHover; border-color: @accentHover; }
QPushButton#primary:pressed { background: @accentPressed; }
QPushButton#primary:disabled { background: @raised; border-color: @raised; color: @faint; }
QPushButton#play {
	background: @accent; border: none; color: #ffffff; font-size: 14pt; font-weight: 700;
	border-radius: 14px; padding: 14px 40px;
}
QPushButton#play:hover { background: @accentHover; }
QPushButton#play:pressed { background: @accentPressed; }
QPushButton#play:disabled { background: @raised; color: @faint; }
QPushButton#danger { color: #ff6b6b; }
QPushButton#danger:hover { background: rgba(255, 107, 107, 0.12); border-color: rgba(255, 107, 107, 0.4); }
QPushButton#danger:disabled { color: @faint; background: @card; border-color: @card; }
QPushButton#swatch { border-radius: 14px; padding: 0; min-width: 28px; max-width: 28px; min-height: 28px; max-height: 28px; }

QLineEdit, QComboBox, QPlainTextEdit, QSpinBox {
	background: @input; border: 1px solid @border; border-radius: 9px; padding: 7px 10px; color: @text;
}
QLineEdit:focus, QComboBox:focus, QPlainTextEdit:focus { border-color: @accentLine; }
QComboBox::drop-down { border: none; width: 28px; }
QComboBox::down-arrow { image: url(@arrow); width: 12px; height: 12px; margin-right: 10px; }
QComboBox QAbstractItemView { background: @raised; border: 1px solid @border; border-radius: 8px; padding: 4px; outline: none; }
QPlainTextEdit#log { font-family: monospace; font-size: 9pt; color: @muted; }

QListWidget, QTableWidget {
	background: @input; border: 1px solid @border; border-radius: 12px; padding: 6px; outline: none;
	alternate-background-color: @card; gridline-color: transparent;
}
QListWidget::item { border-radius: 8px; padding: 10px; margin: 2px 0; }
QListWidget::item:hover { background: @raised; }
QListWidget::item:selected { background: @accentSoft; color: #ffffff; }
QTableWidget::item { padding: 6px; border-radius: 6px; }
QTableWidget::item:selected { background: @accentSoft; color: #ffffff; }
QHeaderView { background: transparent; }
QHeaderView::section {
	background: transparent; color: @faint; border: none; border-bottom: 1px solid @border;
	padding: 8px 6px; font-size: 8pt; font-weight: 700;
}
QTableCornerButton::section { background: transparent; border: none; }

QCheckBox, QRadioButton { spacing: 10px; background: transparent; }
QCheckBox::indicator, QRadioButton::indicator { width: 18px; height: 18px; border: 2px solid @hover; background: @input; }
QCheckBox::indicator { border-radius: 5px; }
QRadioButton::indicator { border-radius: 11px; }
QCheckBox::indicator:checked { background: @accent; border-color: @accent; image: url(@check); }
QRadioButton::indicator:checked { background: @accent; border-color: @accent; }
QCheckBox::indicator:hover, QRadioButton::indicator:hover { border-color: @accentLine; }

QProgressBar { background: @input; border: none; border-radius: 5px; height: 10px; text-align: center; color: transparent; }
QProgressBar::chunk { background: @accent; border-radius: 5px; }

QScrollBar:vertical { background: transparent; width: 10px; margin: 2px; }
QScrollBar:horizontal { background: transparent; height: 10px; margin: 2px; }
QScrollBar::handle { background: @hover; border-radius: 4px; min-height: 24px; min-width: 24px; }
QScrollBar::handle:hover { background: @faint; }
QScrollBar::add-line, QScrollBar::sub-line { width: 0; height: 0; }
QScrollBar::add-page, QScrollBar::sub-page { background: transparent; }
QScrollArea { background: transparent; border: none; }
QScrollArea > QWidget > QWidget { background: transparent; }

QMenu { background: @raised; border: 1px solid @border; border-radius: 8px; padding: 4px; }
QMenu::item { padding: 6px 18px; border-radius: 6px; }
QMenu::item:selected { background: @accentSoft; }

#headerBar { background: @window; }
#headerTitle { font-size: 11pt; font-weight: 700; }
QPushButton#flatIcon { background: transparent; border: none; border-radius: 8px; padding: 6px; }
QPushButton#flatIcon:hover { background: @card; }
QPushButton#flatIcon:pressed { background: @raised; }
QPushButton#flatIcon::menu-indicator { image: none; width: 0; }
QPushButton#closeIcon { background: @card; border: none; border-radius: 13px; padding: 0; min-width: 26px; max-width: 26px; min-height: 26px; max-height: 26px; }
QPushButton#closeIcon:hover { background: @raised; }
#groupTitle { font-size: 11pt; font-weight: 700; }
QFrame#boxed { background: @card; border: none; border-radius: 12px; }
QFrame#rowSeparator { background: @window; border: none; min-height: 1px; max-height: 1px; }
#row, #rowLink, #rowLinkTop, #rowLinkBottom, #rowLinkOnly { background: transparent; }
#rowLink:hover { background: rgba(255, 255, 255, 0.04); }
#rowLinkTop:hover { background: rgba(255, 255, 255, 0.04); border-top-left-radius: 12px; border-top-right-radius: 12px; }
#rowLinkBottom:hover { background: rgba(255, 255, 255, 0.04); border-bottom-left-radius: 12px; border-bottom-right-radius: 12px; }
#rowLinkOnly:hover { background: rgba(255, 255, 255, 0.04); border-radius: 12px; }
#rowTitle { font-size: 10.5pt; }
#rowSubtitle { color: @muted; font-size: 9pt; }
#rowValue { color: @muted; }
QPushButton#iconButton { background: @raised; border: none; border-radius: 8px; padding: 0; min-width: 34px; max-width: 34px; min-height: 34px; max-height: 34px; }
QPushButton#iconButton:hover { background: @hover; }
QPushButton#pill { background: @accent; border: none; border-radius: 17px; padding: 8px 22px; color: #ffffff; font-weight: 700; }
QPushButton#pill:hover { background: @accentHover; }
QPushButton#pill:pressed { background: @accentPressed; }
QPushButton#pill:disabled { background: @raised; color: @faint; }
QPushButton#pillPlain { background: @raised; border: none; border-radius: 17px; padding: 8px 18px; font-weight: 600; }
QPushButton#pillPlain:hover { background: @hover; }
QPushButton#pillPlain:disabled { color: @faint; }
QComboBox#flat { background: transparent; border: none; border-radius: 8px; padding: 6px 4px 6px 10px; }
QComboBox#flat:hover { background: @raised; }
QComboBox#flat::drop-down { width: 22px; }
QComboBox#flat::down-arrow { margin-right: 6px; }
QCheckBox#switch { spacing: 0; }
QCheckBox#switch::indicator { width: 46px; height: 26px; border: none; border-radius: 0; background: transparent; image: url(@switchOff); }
QCheckBox#switch::indicator:checked { image: url(@switchOn); }
#footer { color: @faint; font-size: 8.5pt; }
QLabel:disabled, QComboBox:disabled { color: @faint; }
`)
}

// systemStyle is the desktop's Qt style, remembered before Everblue switches to Fusion
var systemStyle string

// systemSheet keeps the type scale and pills when following the desktop theme, but none of the colors
const systemSheet = `
#brand { font-size: 15pt; font-weight: 700; }
#brandSub, #faint, #sidebarFooter { font-size: 8.5pt; }
#h1 { font-size: 24pt; font-weight: 800; }
#h2 { font-size: 13pt; font-weight: 700; }
#caption { font-size: 8pt; font-weight: 700; letter-spacing: 1px; }
#value { font-size: 12pt; font-weight: 600; }
#mono, QPlainTextEdit#log { font-family: monospace; }
QPushButton#nav { text-align: left; padding: 8px 12px; }
QPushButton#play { font-size: 14pt; font-weight: 700; padding: 12px 36px; }
#pillGood, #pillWarn, #pillAccent { font-weight: 600; }
#headerTitle, #groupTitle { font-size: 11pt; font-weight: 700; }
#rowSubtitle, #footer { font-size: 9pt; }
QFrame#boxed { border: 1px solid palette(mid); border-radius: 10px; }
QFrame#rowSeparator { background: palette(mid); min-height: 1px; max-height: 1px; }
QPushButton#pill { font-weight: 700; padding: 6px 18px; }
`

// applyTheme styles the whole application for the chosen layout and accent, or hands it back to the
// desktop's theme
func applyTheme(app *qt.QApplication, home string, s settings) {
	if systemStyle == "" {
		systemStyle = qt.QApplication_Style().Name()
	}
	takeDesktopColors()
	if s.Theme == themePlain {
		qt.QApplication_SetStyle(qt.QStyleFactory_Create(systemStyle))
		// an empty palette resolves to the platform theme's colors
		qt.QApplication_SetPalette(qt.NewQPalette())
		app.SetStyleSheet(systemSheet)
		return
	}
	qt.QApplication_SetStyle(qt.QStyleFactory_Create("Fusion"))
	a, n := themeAccent(s), colorsFor(s)
	palette := qt.NewQPalette()
	set := func(role qt.QPalette__ColorRole, color string) {
		palette.SetColor2(role, qt.NewQColor6(color))
	}
	set(qt.QPalette__Window, n.window)
	set(qt.QPalette__WindowText, n.text)
	set(qt.QPalette__Base, n.input)
	set(qt.QPalette__AlternateBase, n.card)
	set(qt.QPalette__Text, n.text)
	set(qt.QPalette__Button, n.raised)
	set(qt.QPalette__ButtonText, n.text)
	set(qt.QPalette__ToolTipBase, n.raised)
	set(qt.QPalette__ToolTipText, n.text)
	set(qt.QPalette__PlaceholderText, n.faint)
	set(qt.QPalette__Highlight, a.base)
	set(qt.QPalette__HighlightedText, "#ffffff")
	set(qt.QPalette__Link, a.hover)
	qt.QApplication_SetPalette(palette)
	app.SetStyleSheet(styleSheet(n, a, themeImages(home, n, a)))
}

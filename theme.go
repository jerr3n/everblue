package main

import (
	"fmt"
	"os"
	"strings"

	qt "github.com/mappu/miqt/qt6"
)

// Everblue draws its own dark look rather than following the desktop's Qt theme: Fusion underneath so
// every widget can be restyled, a dark palette for what the stylesheet doesn't reach, and one accent.

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

// The neutrals, darkest first
const (
	colorWindow  = "#0d0f13"
	colorSidebar = "#12151b"
	colorCard    = "#161a21"
	colorRaised  = "#1d222b"
	colorInput   = "#0f1217"
	colorBorder  = "#252b36"
	colorHover   = "#2b3240"
	colorText    = "#e8eaef"
	colorMuted   = "#8b93a3"
	colorFaint   = "#5d6574"
	colorGood    = "#3ccf8e"
	colorWarn    = "#f5b544"
)

func (a accent) rgba(alpha float64) string {
	return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", a.r, a.g, a.b, alpha)
}

// themeImages writes the few glyphs the stylesheet needs as files, since a stylesheet can only name images
func themeImages(home string) (arrow, check string) {
	dir := fmt.Sprintf("%s/.cache/everblue", home)
	os.MkdirAll(dir, 0o755)
	arrow = dir + "/arrow.svg"
	check = dir + "/check.svg"
	os.WriteFile(arrow, []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 12 12"><path d="M2.5 4.5 6 8l3.5-3.5" fill="none" stroke="`+colorMuted+`" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>`), 0o644)
	os.WriteFile(check, []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 18 18"><path d="M4.5 9.5 7.5 12.5 13.5 5.5" fill="none" stroke="#ffffff" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/></svg>`), 0o644)
	return arrow, check
}

func styleSheet(a accent, arrow, check string) string {
	return strings.NewReplacer(
		"@arrow", arrow, "@check", check,
		"@window", colorWindow, "@sidebar", colorSidebar, "@card", colorCard, "@raised", colorRaised,
		"@input", colorInput, "@border", colorBorder, "@hover", colorHover, "@text", colorText,
		"@muted", colorMuted, "@faint", colorFaint, "@good", colorGood, "@warn", colorWarn,
		"@accentSoft", a.rgba(0.16), "@accentLine", a.rgba(0.55), "@accentHover", a.hover,
		"@accentPressed", a.pressed, "@accent", a.base,
	).Replace(`
QMainWindow, QDialog, QMessageBox { background: @window; }
QWidget { color: @text; selection-background-color: @accent; selection-color: #ffffff; }
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
`)
}

// applyTheme styles the whole application with an accent
func applyTheme(app *qt.QApplication, home, name string) {
	qt.QApplication_SetStyle(qt.QStyleFactory_Create("Fusion"))
	a := accentFor(name)
	palette := qt.NewQPalette()
	set := func(role qt.QPalette__ColorRole, color string) {
		palette.SetColor2(role, qt.NewQColor6(color))
	}
	set(qt.QPalette__Window, colorWindow)
	set(qt.QPalette__WindowText, colorText)
	set(qt.QPalette__Base, colorInput)
	set(qt.QPalette__AlternateBase, colorCard)
	set(qt.QPalette__Text, colorText)
	set(qt.QPalette__Button, colorRaised)
	set(qt.QPalette__ButtonText, colorText)
	set(qt.QPalette__ToolTipBase, colorRaised)
	set(qt.QPalette__ToolTipText, colorText)
	set(qt.QPalette__PlaceholderText, colorFaint)
	set(qt.QPalette__Highlight, a.base)
	set(qt.QPalette__HighlightedText, "#ffffff")
	set(qt.QPalette__Link, a.hover)
	qt.QApplication_SetPalette(palette)
	arrow, check := themeImages(home)
	app.SetStyleSheet(styleSheet(a, arrow, check))
}

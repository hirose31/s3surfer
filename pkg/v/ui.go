package v

import (
	"runtime"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

// View ...
type View struct {
	App   *tview.Application
	Frame *tview.Frame
	Pages *tview.Pages
	List  *tview.List
}

func init() {
	if runtime.GOOS == "windows" && runewidth.IsEastAsian() {
		tview.Borders.Horizontal = '-'
		tview.Borders.Vertical = '|'
		tview.Borders.TopLeft = '+'
		tview.Borders.TopRight = '+'
		tview.Borders.BottomLeft = '+'
		tview.Borders.BottomRight = '+'
		tview.Borders.LeftT = '|'
		tview.Borders.RightT = '|'
		tview.Borders.TopT = '-'
		tview.Borders.BottomT = '-'
		tview.Borders.Cross = '+'
		tview.Borders.HorizontalFocus = '='
		tview.Borders.VerticalFocus = '|'
		tview.Borders.TopLeftFocus = '+'
		tview.Borders.TopRightFocus = '+'
		tview.Borders.BottomLeftFocus = '+'
		tview.Borders.BottomRightFocus = '+'
	}
}

// NewView ...
func NewView() View {
	app := tview.NewApplication()

	list := tview.NewList().
		ShowSecondaryText(false)
	list.SetBorder(true).
		SetTitleAlign(tview.AlignLeft)

	main := tview.NewFlex().
		AddItem(list, 0, 1, true)

	pages := tview.NewPages().
		AddPage("main", main, true, true)

	frame := tview.NewFrame(pages)
	frame.AddText("[::b][↓,j/↑,k][::-] Down/Up [::b][Enter,l/u,h][::-] Lower/Upper [::b][d[][::-] Download [::b][q[][::-] Quit", false, tview.AlignCenter, tcell.ColorWhite)

	app.SetRoot(frame, true)

	v := View{
		app,
		frame,
		pages,
		list,
	}

	return v
}

package v

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type View struct {
	App   *tview.Application
	Frame *tview.Frame
	Pages *tview.Pages
	List  *tview.List
}

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

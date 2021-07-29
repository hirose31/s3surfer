package v

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type View struct {
	App  *tview.Application
	List *tview.List
}

func NewView() View {
	app := tview.NewApplication()

	list := tview.NewList().
		ShowSecondaryText(false)
	list.SetBorder(true)

	main := tview.NewFlex().
		AddItem(list, 0, 1, true)

	pages := tview.NewPages().
		AddPage("main", main, true, true)

	frame := tview.NewFrame(pages)
	frame.AddText("Header left", true, tview.AlignLeft, tcell.ColorWhite)
	frame.AddText("Footer center", false, tview.AlignCenter, tcell.ColorWhite)

	app.SetRoot(frame, true)

	v := View{
		app,
		list,
	}

	return v
}

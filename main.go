package main

import (
	"context"
	"log"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type View interface {
	Grid() *ui.Grid
	HandleEvent(context.Context, ui.Event, chan<- View) error
}

type NamedView struct {
	Next *NamedView
	nameParagraph *widgets.Paragraph
}

func NewNamedView(name string) *NamedView {
	var v NamedView

	v.nameParagraph = widgets.NewParagraph()
	v.nameParagraph.Title = name

	return &v
}

func (v *NamedView) Grid() *ui.Grid {
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewRow(1.0/2,
			ui.NewCol(1.0/2, v.nameParagraph),
		),
	)

	return grid
}

func (v *NamedView) HandleEvent(ctx context.Context, e ui.Event, ch chan<- View) error {
	if e.ID == "n" {
		ch <- v.Next
		return nil
	}
	
	if v.nameParagraph.Text == "" {
		v.nameParagraph.Text = e.ID
	} else {
		v.nameParagraph.Text += e.ID
	}

	return nil
}

func runView(ctx context.Context, view View, events <-chan ui.Event, next chan<- View) error {
	ticker := time.NewTicker(10*time.Millisecond)
	defer ticker.Stop()

	grid := view.Grid()
	
	ui.Clear()
	ui.Render(grid)
	for {
		select {
		case e := <-events:
			switch e.ID {
			case "q", "<C-c>":
				next <- nil
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Clear()
				ui.Render(grid)
			default:
				view.HandleEvent(ctx, e, next)
			}
		case <-ticker.C:
			ui.Clear()
			ui.Render(grid)
		case <-ctx.Done():
			return nil
		}
	}
}

func main() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	events := ui.PollEvents()
	ctx, cancel := context.WithCancel(context.Background())

	firstView := NewNamedView("First view")
	secondView := NewNamedView("Second view")

	firstView.Next = secondView
	secondView.Next = firstView

	views := make(chan View)

	go func() {
		views <- firstView
	}()

	for view := range views {
		cancel()

		if view == nil {
			return
		}

		ctx, cancel = context.WithCancel(context.Background())

		go func(v View) {
			runView(ctx, v, events, views)
		}(view)
	}
}

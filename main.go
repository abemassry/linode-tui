package main

import (
	"context"
	"log"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"

	"github.com/linode/linodego"
	"golang.org/x/oauth2"

	"net/http"
	"os"
)

type View interface {
	Grid() *ui.Grid
	HandleEvent(context.Context, ui.Event, chan<- View) error
}

type NamedView struct {
	Next          View
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

func getLinodes() []linodego.Instance {
	apiKey, ok := os.LookupEnv("LINODE_TOKEN")
	if !ok {
		log.Fatal("Could not find LINODE_TOKEN, please assert it is set.")
	}
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apiKey})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	linodeClient := linodego.NewClient(oauth2Client)
	linodeClient.SetDebug(false)

	linodes, err := linodeClient.ListInstances(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	return linodes
}

type LinodeDetailView struct {
	linode             linodego.Instance
	parentView         View
	tableWidget        *widgets.Table
	instructionsWidget *widgets.Paragraph
}

func NewLinodeDetailView(parent View, i linodego.Instance) *LinodeDetailView {
	tableWidget := widgets.NewTable()

	tableWidget.Rows = [][]string{
		[]string{"Label", i.Label},
		[]string{"Status", string(i.Status)},
		[]string{"Plan", i.Type},
		[]string{"IP", i.IPv4[0].String()},
		[]string{"Location", i.Region},
	}
	tableWidget.Title = "Details"

	instructionsWidget := widgets.NewParagraph()
	instructionsWidget.Text = "Available actions: (b)oot, (s)hutdown, (l)ist"
	instructionsWidget.Title = "Instructions"

	v := &LinodeDetailView{
		linode:             i,
		parentView:         parent,
		tableWidget:        tableWidget,
		instructionsWidget: instructionsWidget,
	}

	return v
}

func (v *LinodeDetailView) Grid() *ui.Grid {
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(.75,
			ui.NewCol(1.0, v.tableWidget),
		),
		ui.NewRow(.25,
			ui.NewCol(1.0, v.instructionsWidget),
		),
	)
	return grid
}

func (v *LinodeDetailView) HandleEvent(ctx context.Context, e ui.Event, ch chan<- View) error {
	switch e.ID {
	case "l":
		ch <- v.parentView
	default:
	}

	return nil
}

type LinodesView struct {
	linodes     []linodego.Instance
	linodeViews []*LinodeDetailView
	listWidget  *widgets.List
}

func NewLinodesView() *LinodesView {
	w := widgets.NewList()
	w.Title = "Linodes"
	linodes := getLinodes()
	for _, linode := range linodes {
		w.Rows = append(w.Rows, linode.Label)
	}
	w.TextStyle = ui.NewStyle(ui.ColorBlue)
	w.WrapText = false

	v := &LinodesView{
		linodes:    linodes,
		listWidget: w,
	}

	views := make([]*LinodeDetailView, len(linodes))
	for i, linode := range linodes {
		views[i] = NewLinodeDetailView(v, linode)
	}

	v.linodeViews = views

	return v
}

func (v *LinodesView) Grid() *ui.Grid {
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(1.0/2,
			ui.NewCol(1.0/2, v.listWidget),
		),
	)
	return grid
}

func (v *LinodesView) HandleEvent(ctx context.Context, e ui.Event, next chan<- View) error {
	switch e.ID {
	case "j", "<Down>":
		v.listWidget.ScrollDown()
	case "k", "<Up>":
		v.listWidget.ScrollUp()
	case "<C-d>":
		v.listWidget.ScrollHalfPageDown()
	case "<C-u>":
		v.listWidget.ScrollHalfPageUp()
	case "<C-f>":
		v.listWidget.ScrollPageDown()
	case "<C-b>":
		v.listWidget.ScrollPageUp()
	case "<Home>":
		v.listWidget.ScrollTop()
	case "G", "<End>":
		v.listWidget.ScrollBottom()
	case "<Enter>":
		idx := v.listWidget.SelectedRow
		next <- v.linodeViews[idx]
	default:
	}

	return nil
}

func runView(ctx context.Context, view View, events <-chan ui.Event, next chan<- View) error {
	ticker := time.NewTicker(10 * time.Millisecond)
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

	linodesView := NewLinodesView()

	// secondView.Next = linodesView

	views := make(chan View)

	go func() {
		views <- linodesView
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

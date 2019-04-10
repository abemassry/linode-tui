package tui

import (
	"context"
	"fmt"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/linode/linodego"
	"strings"
	"sync"
	"time"
)

var (
	FinalView = fmt.Errorf("final view")
)

// View represents a terminal-level view in the TUI. Views may dynamically update in the background;
// however, widget state should not be updated while the view mutex is locked.
type View interface {
	// Initialize sets up the view and returns a grid of UI elements to render in the terminal.
	// Views are assumed to exist for the duration of the provided context, and should call the
	// provided render function when they should be re-rendered.
	Initialize(context.Context, func()) (*ui.Grid, error)

	// HandleEvent is called when a UI event that is not a quit or resize event is found.
	// When the view is ready to hand control to another view, it should return that View as
	// (view, nil). Otherwise, it should return (nil, nil). If this view is the final view,
	// it should return (nil, FinalView).
	HandleEvent(context.Context, ui.Event) (View, error)

	sync.Locker
}

// RunView runs a single view. Views are locked before any rendering occurs and unlocked after
// rendering finishes. Events are handled as follows:
// - If the event is the literal "q" or "<C-c>", a nil value will be written to the next view
//   channel.
// - If the event is a resize, the grid will be resized and the UI rerendered.
// - Otherwise, the event will be passed to the view's HandleEvent method.
func RunView(ctx context.Context, view View, events <-chan ui.Event) (View, error) {
	renderCh := make(chan struct{}, 1)

	render := func() {
		select {
		case renderCh <- struct{}{}:
		default:
		}
	}

	grid, errInitialize := view.Initialize(ctx, render)
	if errInitialize != nil {
		return nil, errInitialize
	}

	ui.Clear()
	ui.Render(grid)
	for {
		select {
		case e := <-events:
			switch e.ID {
			case "q", "<C-c>":
				return nil, FinalView
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				view.Lock()
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Clear()
				ui.Render(grid)
				view.Unlock()
			default:
				next, errEvent := view.HandleEvent(ctx, e)
				switch {
				case next == nil && errEvent == nil:
				case next == nil && errEvent == FinalView:
					return nil, nil
				case next != nil && errEvent == nil:
					return next, nil
				case next != nil && errEvent == nil:
					return next, fmt.Errorf("next view defined for final view")
				default:
					return next, errEvent
				}
			}
		case <-renderCh:
			view.Lock()
			ui.Clear()
			ui.Render(grid)
			view.Unlock()
		case <-ctx.Done():
			return nil, FinalView
		}
	}
}

type LinodesView struct {
	client              *linodego.Client
	linodes             []linodego.Instance
	notifications       []linodego.Notification
	linodesWidget       *widgets.List
	notificationsWidget *widgets.List
	tabsWidget          *widgets.TabPane
	render              func()

	sync.Mutex
}

func NewLinodesView(client *linodego.Client) *LinodesView {
	return &LinodesView{
		client: client,
	}
}

func (v *LinodesView) initialize(ctx context.Context, render func()) error {
	v.linodesWidget = widgets.NewList()
	v.linodesWidget.Title = "Linodes"
	v.linodesWidget.TextStyle = ui.NewStyle(ui.ColorBlue)
	v.linodesWidget.WrapText = false

	v.notificationsWidget = widgets.NewList()
	v.notificationsWidget.Title = "Notifications"
	v.linodesWidget.TextStyle = ui.NewStyle(ui.ColorBlue)
	v.linodesWidget.WrapText = true

	v.tabsWidget = widgets.NewTabPane("Linodes", "NodeBalancers", "DNS Manager", "Account", "Support", "My Profile")

	v.render = render

	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				v.updateState(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	return v.updateState(ctx)
}

func (v *LinodesView) grid() *ui.Grid {
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(1.0/16, v.tabsWidget),
		ui.NewRow(1.0/2,
			ui.NewCol(1.0/2, v.linodesWidget),
			ui.NewCol(1.0/2, v.notificationsWidget),
		),
	)
	return grid
}

func (v *LinodesView) updateState(ctx context.Context) error {
	defer v.render()

	linodes, errLinodes := v.client.ListInstances(ctx, nil)
	if errLinodes != nil {
		return errLinodes
	}

	notifications, errNotifications := v.client.ListNotifications(ctx, nil)
	if errNotifications != nil {
		return errNotifications
	}

	v.Lock()
	defer v.Unlock()

	v.linodes = linodes
	v.notifications = notifications

	v.linodesWidget.Rows = make([]string, len(linodes))
	for i, linode := range v.linodes {
		v.linodesWidget.Rows[i] = fmt.Sprintf("%s (%s, %s)", linode.Label, linode.Type, linode.Status)
	}

	v.notificationsWidget.Rows = make([]string, len(notifications))
	for i, notification := range v.notifications {
		v.notificationsWidget.Rows[i] = notification.Label
	}

	return nil
}

func (v *LinodesView) Initialize(ctx context.Context, render func()) (*ui.Grid, error) {
	if errInitialize := v.initialize(ctx, render); errInitialize != nil {
		return nil, errInitialize
	}

	return v.grid(), nil
}

func (v *LinodesView) HandleEvent(ctx context.Context, e ui.Event) (View, error) {
	defer v.render()

	switch e.ID {
	case "j", "<Down>":
		v.linodesWidget.ScrollDown()
	case "k", "<Up>":
		v.linodesWidget.ScrollUp()
	case "h", "<Left>":
		v.tabsWidget.FocusLeft()
	case "l", "<Right>":
		v.tabsWidget.FocusRight()
	case "<C-d>":
		v.linodesWidget.ScrollHalfPageDown()
	case "<C-u>":
		v.linodesWidget.ScrollHalfPageUp()
	case "<C-f>":
		v.linodesWidget.ScrollPageDown()
	case "<C-b>":
		v.linodesWidget.ScrollPageUp()
	case "<Home>":
		v.linodesWidget.ScrollTop()
	case "G", "<End>":
		v.linodesWidget.ScrollBottom()
	case "<Enter>":
		idx := v.linodesWidget.SelectedRow
		next := NewLinodeDetailView(v.client, v, &v.linodes[idx])
		return next, nil
	default:
	}

	return nil, nil
}

type LinodeDetailView struct {
	client             *linodego.Client
	instance           *linodego.Instance
	parentView         View
	tableWidget        *widgets.Table
	instructionsWidget *widgets.Paragraph
	render             func()

	sync.Mutex
}

func NewLinodeDetailView(client *linodego.Client, parent View, instance *linodego.Instance) *LinodeDetailView {
	v := LinodeDetailView{
		client:     client,
		parentView: parent,
		instance:   instance,
	}

	return &v
}

func (v *LinodeDetailView) updateState(ctx context.Context) error {
	instance, errInstance := v.client.GetInstance(ctx, v.instance.ID)
	if errInstance != nil {
		return errInstance
	}

	v.instance = instance

	v.renderState()

	return nil
}

func (v *LinodeDetailView) renderState() {
	defer v.render()

	v.Lock()
	defer v.Unlock()

	ipv4Strs := make([]string, len(v.instance.IPv4))
	for i, ip := range v.instance.IPv4 {
		ipv4Strs[i] = ip.String()
	}
	ipv4sStr := strings.Join(ipv4Strs, ", ")

	v.tableWidget.Rows = [][]string{
		[]string{"Label", v.instance.Label},
		[]string{"Status", string(v.instance.Status)},
		[]string{"Plan", v.instance.Type},
		[]string{"IPv4", ipv4sStr},
		[]string{"Location", v.instance.Region},
	}
	v.tableWidget.ColumnWidths = []int{15, 500}
	v.tableWidget.Title = "Details"
	switch v.instance.Status {
	case "offline":
		v.tableWidget.RowStyles[1] = ui.NewStyle(ui.ColorWhite, ui.ColorRed, ui.ModifierBold)
	case "booting", "shutting_down":
		v.tableWidget.RowStyles[1] = ui.NewStyle(ui.ColorBlack, ui.ColorYellow, ui.ModifierBold)
	case "running":
		v.tableWidget.RowStyles[1] = ui.NewStyle(ui.ColorWhite, ui.ColorGreen, ui.ModifierBold)
	}
}

func (v *LinodeDetailView) grid() *ui.Grid {
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

func (v *LinodeDetailView) Initialize(ctx context.Context, render func()) (*ui.Grid, error) {
	v.tableWidget = widgets.NewTable()
	v.tableWidget.ColumnWidths = []int{15}
	v.tableWidget.Title = "Details"

	v.instructionsWidget = widgets.NewParagraph()
	v.instructionsWidget.Text = "Available actions: (b)oot, (s)hutdown, (l)ist"
	v.instructionsWidget.Title = "Instructions"

	v.render = render

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				v.updateState(ctx)
				v.renderState()
			case <-ctx.Done():
				return
			}
		}
	}()

	grid := v.grid()

	v.renderState()

	return grid, nil
}

func (v *LinodeDetailView) HandleEvent(ctx context.Context, e ui.Event) (View, error) {
	defer v.render()

	switch e.ID {
	case "l":
		return v.parentView, nil
	case "b":
		v.bootInstance(ctx)
	case "s":
		v.shutdownInstance(ctx)
	default:
	}

	return nil, nil
}

func (v *LinodeDetailView) bootInstance(ctx context.Context) error {
	return v.client.BootInstance(ctx, v.instance.ID, 0)
}

func (v *LinodeDetailView) shutdownInstance(ctx context.Context) error {
	return v.client.ShutdownInstance(ctx, v.instance.ID)
}

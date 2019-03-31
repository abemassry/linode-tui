package main

import (
	"context"
	"flag"
	"github.com/abemassry/linode-tui"
	ui "github.com/gizak/termui/v3"
	"github.com/linode/linodego"
	"golang.org/x/oauth2"
	"log"

	"net/http"
	"os"
)

func setLightTheme() {
	ui.Theme.Default = ui.NewStyle(ui.ColorBlack)
	ui.Theme.Block.Title = ui.NewStyle(ui.ColorBlack)
	ui.Theme.Block.Border = ui.NewStyle(ui.ColorBlack)
	ui.Theme.List.Text = ui.NewStyle(ui.ColorBlack)
	ui.Theme.Table.Text = ui.NewStyle(ui.ColorBlack)
	ui.Theme.Paragraph.Text = ui.NewStyle(ui.ColorBlack)
}

func main() {
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

	var lightTheme bool
	flag.BoolVar(&lightTheme, "lightTheme", false, "use light theme")

	flag.Parse()

	if lightTheme {
		setLightTheme()
	}

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	events := ui.PollEvents()
	ctx, cancel := context.WithCancel(context.Background())

	var view tui.View

	view = tui.NewLinodesView(&linodeClient)

	for {
		next, errView := tui.RunView(ctx, view, events)
		cancel()
		switch errView {
		case nil:
			view = next
		case tui.FinalView:
			return
		default:
			log.Fatalf("view error: %s", errView.Error())
		}

		ctx, cancel = context.WithCancel(context.Background())
	}
}

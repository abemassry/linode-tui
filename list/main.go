// Copyright 2017 Zack Guo <zack.y.guo@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

// +build ignore

package main

import (
	"log"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"

	"context"
	"github.com/linode/linodego"
	"golang.org/x/oauth2"

	"net/http"
	"os"
)

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

func main() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	linodes := getLinodes()
	linodeStrs := make([]string, len(linodes))
	for i, l := range linodes {
		linodeStrs[i] = l.Label
	}

	l := widgets.NewList()
	l.Title = "Linodes"
	l.Rows = []string{
		linodeStrs[0],
		linodeStrs[1],
		linodeStrs[2],
		linodeStrs[3],
		linodeStrs[4],
		linodeStrs[5],
		linodeStrs[6],
	}
	l.TextStyle = ui.NewStyle(ui.ColorGreen)
	l.WrapText = false
	l.SetRect(0, 0, 60, 15)

	ui.Render(l)
	m := widgets.NewList()
	m.Title = "Linodes"
	m.Rows = []string{
		"linode1234 IP region",
		"boot",
		"shutdown",
	}
	m.TextStyle = ui.NewStyle(ui.ColorBlue)
	m.WrapText = false
	m.SetRect(0, 0, 60, 15)
	i := 0

	previousKey := ""
	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		if i == 0 {
			switch e.ID {
			case "q", "<C-c>":
				return
			case "j", "<Down>":
				l.ScrollDown()
			case "k", "<Up>":
				l.ScrollUp()
			case "<C-d>":
				l.ScrollHalfPageDown()
			case "<C-u>":
				l.ScrollHalfPageUp()
			case "<C-f>":
				l.ScrollPageDown()
			case "<C-b>":
				l.ScrollPageUp()
			case "g":
				if previousKey == "g" {
					l.ScrollTop()
				}
			case "<Home>":
				l.ScrollTop()
			case "G", "<End>":
				l.ScrollBottom()
			case "m":
				i = 1
			}

			if previousKey == "g" {
				previousKey = ""
			} else {
				previousKey = e.ID
			}

			ui.Render(l)
		} else {
			switch e.ID {
			case "q", "<C-c>":
				return
			case "j", "<Down>":
				m.ScrollDown()
			case "k", "<Up>":
				m.ScrollUp()
			case "<C-d>":
				m.ScrollHalfPageDown()
			case "<C-u>":
				m.ScrollHalfPageUp()
			case "<C-f>":
				m.ScrollPageDown()
			case "<C-b>":
				m.ScrollPageUp()
			case "g":
				if previousKey == "g" {
					m.ScrollTop()
				}
			case "<Home>":
				m.ScrollTop()
			case "G", "<End>":
				m.ScrollBottom()
			case "m":
				i = 0
			}

			if previousKey == "g" {
				previousKey = ""
			} else {
				previousKey = e.ID
			}

			ui.Render(m)
		}
	}
}

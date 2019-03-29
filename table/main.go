// Copyright 2017 Zack Guo <zack.y.guo@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

// +build ignore

package main

import (
	"log"

	"context"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"

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

	table1 := widgets.NewTable()

	linodes := getLinodes()
	linodeStrs := make([]string, len(linodes))
	for i, l := range linodes {
		linodeStrs[i] = l.Label
	}

	table1.Rows = [][]string{
		[]string{"Linode Name", "Location", "IP", "Status"},
		[]string{linodeStrs[0], "Go-lang is so cool", "Im working on Ruby"},
		[]string{linodeStrs[1], "10", "11"},
		[]string{linodeStrs[2], "10", "11"},
		[]string{linodeStrs[3], "10", "11"},
		[]string{linodeStrs[4], "10", "11"},
	}
	table1.TextStyle = ui.NewStyle(ui.ColorWhite)
	table1.SetRect(0, 0, 60, 10)

	ui.Render(table1)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		}
	}
}

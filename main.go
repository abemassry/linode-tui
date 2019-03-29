package main

import (
	"context"
	"log"
	"time"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/termbox"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/button"
	"github.com/mum4k/termdash/widgets/text"
)

// LinodeView represents an entire screen view of the TUI. Views should, when ready to pass control
// to another view, send the next view to be rendered to the send-only LinodeView channel. If the
// next state is terminal (i.e. a quit event), the view should send nil instead.
type LinodeView func(context.Context, *termbox.Terminal, chan<- LinodeView) error

// firstView represents the first view of this demo. Like secondView, it just shows the text "First
// view" and a button to go to secondView.
func firstView(ctx context.Context, t *termbox.Terminal, out chan<- LinodeView) error {
	display, err := text.New()
	if err != nil {
		return err
	}

	err = display.Write("first view")
	if err != nil {
		return err
	}

	nextButton, err := button.New("second view", func() error {
		out <- secondView

		return nil
	})
	if err != nil {
		return err
	}

	c, err := container.New(
		t,
		container.Border(linestyle.Light),
		container.BorderTitle("FIRST VIEW - PRESS Q TO QUIT"),
		container.SplitHorizontal(
			container.Top(
				container.PlaceWidget(display),
			),
			container.Bottom(
				container.PlaceWidget(nextButton),
			),
			container.SplitPercent(60),
		),
	)

	if err != nil {
		return err
	}

	// If the user hits q or Q, send a nil view to the state machine runner
	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == 'q' || k.Key == 'Q' {
			out <- nil
		}
	}

	errRun := termdash.Run(
		ctx,
		t,
		c,
		termdash.KeyboardSubscriber(quitter),
		termdash.RedrawInterval(100*time.Millisecond),
	)

	return errRun
}

// secondView represents the second view of this demo. Like firstView, it just shows the text
// "Second view" and a button to go to firstView.
func secondView(ctx context.Context, t *termbox.Terminal, out chan<- LinodeView) error {
	display, err := text.New()
	if err != nil {
		return err
	}

	err = display.Write("second view")
	if err != nil {
		return err
	}

	nextButton, err := button.New("first view", func() error {
		out <- firstView

		return nil
	})
	if err != nil {
		return err
	}

	c, err := container.New(
		t,
		container.Border(linestyle.Light),
		container.BorderTitle("SECOND VIEW - PRESS Q TO QUIT"),
		container.SplitHorizontal(
			container.Top(
				container.PlaceWidget(display),
			),
			container.Bottom(
				container.PlaceWidget(nextButton),
			),
			container.SplitPercent(60),
		),
	)

	if err != nil {
		return err
	}

	// If the user hits q or Q, send a nil view to the state machine runner
	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == 'q' || k.Key == 'Q' {
			out <- nil
		}
	}

	errRun := termdash.Run(
		ctx,
		t,
		c,
		termdash.KeyboardSubscriber(quitter),
		termdash.RedrawInterval(100*time.Millisecond),
	)

	return errRun
}

func main() {
	// Create a new terminal, or fail.
	t, err := termbox.New()
	if err != nil {
		log.Fatal(err)
	}

	// Mother says good children close their resources.
	defer t.Close()

	// Set up a new context.
	ctx, cancel := context.WithCancel(context.Background())

	// Set up a the channel of views...
	views := make(chan LinodeView)

	// ...and prime the channel with a deferred send to the first view.
	//
	// And AC said, "LET THERE BE LIGHT!" 
	go func() {
		views <- firstView
	}()

	// And there was light----
	//
	// Iterate over the channel, canceling each obsolete view's context and instantiating a new
	// context in its place.
	for view := range views {
		// This function comes from up above, or from below, depending on your point of
		// view.
		cancel()

		// If the next view is nil, we're done. Break out of the loop and exit gracefully
		if view == nil {
			return
		}

		// Instantiate a new context for the new view.
		ctx, cancel = context.WithCancel(context.Background())

		// Run the new view in its own goroutine so we aren't blocked. Using a closure to
		// capture the current view avoids potential issues we might have with concurrent
		// execution, since goroutines take a while to start.
		go func(v LinodeView) {
			err := v(ctx, t, views)
			if err != nil {
				log.Printf("error in view: %s", err.Error())
			}
		}(view)
	}
}

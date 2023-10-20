// Package termui contains the interactive terminal UI
package termui

import (
	"fmt"

	"github.com/awesome-gocui/gocui"
	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/cache"
	buginput "github.com/MichaelMure/git-bug/commands/bug/input"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/query"
	"github.com/MichaelMure/git-bug/util/text"
)

var errTerminateMainloop = errors.New("terminate gocui mainloop")

type termUI struct {
	g      *gocui.Gui
	gError chan error
	cache  *cache.RepoCache

	activeWindow window

	bugTable    *bugTable
	showBug     *showBug
	labelSelect *labelSelect
	msgPopup    *msgPopup
	inputPopup  *inputPopup
}

func (tui *termUI) activateWindow(window window) error {
	if err := tui.activeWindow.disable(tui.g); err != nil {
		return err
	}

	tui.activeWindow = window

	return nil
}

var ui *termUI

type window interface {
	keybindings(g *gocui.Gui) error
	layout(g *gocui.Gui) error
	disable(g *gocui.Gui) error
}

// Run will launch the termUI in the terminal
func Run(cache *cache.RepoCache) error {
	ui = &termUI{
		gError:      make(chan error, 1),
		cache:       cache,
		bugTable:    newBugTable(cache),
		showBug:     newShowBug(cache),
		labelSelect: newLabelSelect(),
		msgPopup:    newMsgPopup(),
		inputPopup:  newInputPopup(),
	}

	ui.activeWindow = ui.bugTable

	initGui(nil)

	err := <-ui.gError

	type errorStack interface {
		ErrorStack() string
	}

	if err != nil && err != gocui.ErrQuit {
		if e, ok := err.(errorStack); ok {
			fmt.Println(e.ErrorStack())
		}
		return err
	}

	return nil
}

func initGui(action func(ui *termUI) error) {
	g, err := gocui.NewGui(gocui.Output256, false)

	if err != nil {
		ui.gError <- err
		return
	}

	ui.g = g

	ui.g.SetManagerFunc(layout)

	ui.g.InputEsc = true

	err = keybindings(ui.g)

	if err != nil {
		ui.g.Close()
		ui.g = nil
		ui.gError <- err
		return
	}

	if action != nil {
		err = action(ui)
		if err != nil {
			ui.g.Close()
			ui.g = nil
			ui.gError <- err
			return
		}
	}

	err = g.MainLoop()

	if err != nil && err != errTerminateMainloop {
		if ui.g != nil {
			ui.g.Close()
		}
		ui.gError <- err
	}
}

func layout(g *gocui.Gui) error {
	g.Cursor = false

	if err := ui.activeWindow.layout(g); err != nil {
		return err
	}

	if err := ui.msgPopup.layout(g); err != nil {
		return err
	}

	if err := ui.inputPopup.layout(g); err != nil {
		return err
	}

	return nil
}

func keybindings(g *gocui.Gui) error {
	// Quit
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	if err := ui.bugTable.keybindings(g); err != nil {
		return err
	}

	if err := ui.showBug.keybindings(g); err != nil {
		return err
	}

	if err := ui.labelSelect.keybindings(g); err != nil {
		return err
	}

	if err := ui.msgPopup.keybindings(g); err != nil {
		return err
	}

	if err := ui.inputPopup.keybindings(g); err != nil {
		return err
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func newBugWithEditor(repo *cache.RepoCache) error {
	// This is somewhat hacky.
	// As there is no way to pause gocui, run the editor and restart gocui,
	// we have to stop it entirely and start a new one later.
	//
	// - an error channel is used to route the returned error of this new
	// 		instance into the original launch function
	// - a custom error (errTerminateMainloop) is used to terminate the original
	//		instance's mainLoop. This error is then filtered.

	ui.g.Close()
	ui.g = nil

	title, message, err := buginput.BugCreateEditorInput(ui.cache, "", "")

	if err != nil && err != buginput.ErrEmptyTitle {
		return err
	}

	var b *cache.BugCache
	if err == buginput.ErrEmptyTitle {
		ui.msgPopup.Activate(msgPopupErrorTitle, "Empty title, aborting.")
		initGui(nil)

		return errTerminateMainloop
	} else {
		b, _, err = repo.Bugs().New(
			text.CleanupOneLine(title),
			text.Cleanup(message),
		)
		if err != nil {
			return err
		}

		initGui(func(ui *termUI) error {
			ui.showBug.SetBug(b)
			return ui.activateWindow(ui.showBug)
		})

		return errTerminateMainloop
	}
}

func addCommentWithEditor(bug *cache.BugCache) error {
	// This is somewhat hacky.
	// As there is no way to pause gocui, run the editor and restart gocui,
	// we have to stop it entirely and start a new one later.
	//
	// - an error channel is used to route the returned error of this new
	// 		instance into the original launch function
	// - a custom error (errTerminateMainloop) is used to terminate the original
	//		instance's mainLoop. This error is then filtered.

	ui.g.Close()
	ui.g = nil

	message, err := buginput.BugCommentEditorInput(ui.cache, "")
	if err != nil && err != buginput.ErrEmptyMessage {
		return err
	}
	if err == buginput.ErrEmptyMessage {
		ui.msgPopup.Activate(msgPopupErrorTitle, "Empty message, aborting.")
	} else {
		_, _, err := bug.AddComment(text.Cleanup(message))
		if err != nil {
			return err
		}
	}

	initGui(nil)

	return errTerminateMainloop
}

func editCommentWithEditor(bug *cache.BugCache, target entity.CombinedId, preMessage string) error {
	// This is somewhat hacky.
	// As there is no way to pause gocui, run the editor and restart gocui,
	// we have to stop it entirely and start a new one later.
	//
	// - an error channel is used to route the returned error of this new
	// 		instance into the original launch function
	// - a custom error (errTerminateMainloop) is used to terminate the original
	//		instance's mainLoop. This error is then filtered.

	ui.g.Close()
	ui.g = nil

	message, err := buginput.BugCommentEditorInput(ui.cache, preMessage)
	if err != nil && err != buginput.ErrEmptyMessage {
		return err
	}

	if err == buginput.ErrEmptyMessage {
		// TODO: Allow comments to be deleted?
		ui.msgPopup.Activate(msgPopupErrorTitle, "Empty message, aborting.")
	} else if message == preMessage {
		ui.msgPopup.Activate(msgPopupErrorTitle, "No changes found, aborting.")
	} else {
		_, err := bug.EditComment(target, text.Cleanup(message))
		if err != nil {
			return err
		}
	}

	initGui(nil)

	return errTerminateMainloop
}

func setTitleWithEditor(bug *cache.BugCache) error {
	// This is somewhat hacky.
	// As there is no way to pause gocui, run the editor and restart gocui,
	// we have to stop it entirely and start a new one later.
	//
	// - an error channel is used to route the returned error of this new
	// 		instance into the original launch function
	// - a custom error (errTerminateMainloop) is used to terminate the original
	//		instance's mainLoop. This error is then filtered.

	ui.g.Close()
	ui.g = nil

	snap := bug.Compile()

	title, err := buginput.BugTitleEditorInput(ui.cache, snap.Title)

	if err != nil && err != buginput.ErrEmptyTitle {
		return err
	}

	if err == buginput.ErrEmptyTitle {
		ui.msgPopup.Activate(msgPopupErrorTitle, "Empty title, aborting.")
	} else if title == snap.Title {
		ui.msgPopup.Activate(msgPopupErrorTitle, "No change, aborting.")
	} else {
		_, err := bug.SetTitle(text.CleanupOneLine(title))
		if err != nil {
			return err
		}
	}

	initGui(nil)

	return errTerminateMainloop
}

func editQueryWithEditor(bt *bugTable) error {
	// This is somewhat hacky.
	// As there is no way to pause gocui, run the editor and restart gocui,
	// we have to stop it entirely and start a new one later.
	//
	// - an error channel is used to route the returned error of this new
	// 		instance into the original launch function
	// - a custom error (errTerminateMainloop) is used to terminate the original
	//		instance's mainLoop. This error is then filtered.

	ui.g.Close()
	ui.g = nil

	queryStr, err := buginput.QueryEditorInput(bt.repo, bt.queryStr)

	if err != nil {
		return err
	}

	bt.queryStr = queryStr

	q, err := query.Parse(queryStr)

	if err != nil {
		ui.msgPopup.Activate(msgPopupErrorTitle, err.Error())
	} else {
		bt.query = q
	}

	initGui(nil)

	return errTerminateMainloop
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a > b {
		return b
	}
	return a
}

package termui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/awesome-gocui/gocui"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entities/bug"
)

const labelSelectView = "labelSelectView"
const labelSelectInstructionsView = "labelSelectInstructionsView"

var labelSelectHelp = helpBar{
	{"q", "Save and close"},
	{"↓↑,jk", "Nav"},
	{"a", "Add item"},
}

type labelSelect struct {
	cache       *cache.RepoCache
	bug         *cache.BugCache
	labels      []bug.Label
	labelSelect []bool
	selected    int
	scroll      int
	childViews  []string
}

func newLabelSelect() *labelSelect {
	return &labelSelect{}
}

func (ls *labelSelect) SetBug(cache *cache.RepoCache, bug *cache.BugCache) {
	ls.cache = cache
	ls.bug = bug
	ls.labels = cache.Bugs().ValidLabels()

	// Find which labels are currently applied to the bug
	bugLabels := bug.Compile().Labels
	labelSelect := make([]bool, len(ls.labels))
	for i, label := range ls.labels {
		for _, bugLabel := range bugLabels {
			if label == bugLabel {
				labelSelect[i] = true
				break
			}
		}
	}

	ls.labelSelect = labelSelect
	if len(labelSelect) > 0 {
		ls.selected = 0
	} else {
		ls.selected = -1
	}

	ls.scroll = 0
}

func (ls *labelSelect) keybindings(g *gocui.Gui) error {
	// Abort
	if err := g.SetKeybinding(labelSelectView, gocui.KeyEsc, gocui.ModNone, ls.abort); err != nil {
		return err
	}
	// Save and return
	if err := g.SetKeybinding(labelSelectView, 'q', gocui.ModNone, ls.saveAndReturn); err != nil {
		return err
	}
	// Up
	if err := g.SetKeybinding(labelSelectView, gocui.KeyArrowUp, gocui.ModNone, ls.selectPrevious); err != nil {
		return err
	}
	if err := g.SetKeybinding(labelSelectView, 'k', gocui.ModNone, ls.selectPrevious); err != nil {
		return err
	}
	// Down
	if err := g.SetKeybinding(labelSelectView, gocui.KeyArrowDown, gocui.ModNone, ls.selectNext); err != nil {
		return err
	}
	if err := g.SetKeybinding(labelSelectView, 'j', gocui.ModNone, ls.selectNext); err != nil {
		return err
	}
	// Select
	if err := g.SetKeybinding(labelSelectView, gocui.KeySpace, gocui.ModNone, ls.selectItem); err != nil {
		return err
	}
	if err := g.SetKeybinding(labelSelectView, 'x', gocui.ModNone, ls.selectItem); err != nil {
		return err
	}
	if err := g.SetKeybinding(labelSelectView, gocui.KeyEnter, gocui.ModNone, ls.selectItem); err != nil {
		return err
	}
	// Add
	if err := g.SetKeybinding(labelSelectView, 'a', gocui.ModNone, ls.addItem); err != nil {
		return err
	}
	return nil
}

func (ls *labelSelect) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	ls.childViews = nil

	width := 5
	for _, label := range ls.labels {
		width = maxInt(width, len(label.String()))
	}
	width += 10
	x0 := 1
	y0 := 0 - ls.scroll

	v, err := g.SetView(labelSelectView, x0, 0, x0+width, maxY-2, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}

		v.Frame = false
	}

	for i, label := range ls.labels {
		viewname := fmt.Sprintf("labeledit%d", i)
		v, err := g.SetView(viewname, x0+2, y0, x0+width+2, y0+2, 0)
		if err != nil && !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		ls.childViews = append(ls.childViews, viewname)
		v.Frame = i == ls.selected
		v.Clear()
		selectBox := " [ ] "
		if ls.labelSelect[i] {
			selectBox = " [x] "
		}

		lc := label.Color()
		lc256 := lc.Term256()
		labelStr := lc256.Escape() + "◼ " + lc256.Unescape() + label.String()
		_, _ = fmt.Fprint(v, selectBox, labelStr)

		y0 += 2
	}

	v, err = g.SetView(labelSelectInstructionsView, -1, maxY-2, maxX, maxY, 0)
	ls.childViews = append(ls.childViews, labelSelectInstructionsView)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Frame = false
		v.FgColor = gocui.ColorWhite
	}
	v.Clear()
	_, _ = fmt.Fprint(v, labelSelectHelp.Render(maxX))
	if _, err = g.SetViewOnTop(labelSelectInstructionsView); err != nil {
		return err
	}
	if _, err := g.SetCurrentView(labelSelectView); err != nil {
		return err
	}
	return nil
}

func (ls *labelSelect) disable(g *gocui.Gui) error {
	for _, view := range ls.childViews {
		if err := g.DeleteView(view); err != nil && !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
	}
	return nil
}

func (ls *labelSelect) focusView(g *gocui.Gui) error {
	if ls.selected < 0 {
		return nil
	}

	_, lsy0, _, lsy1, err := g.ViewPosition(labelSelectView)
	if err != nil {
		return err
	}

	_, vy0, _, vy1, err := g.ViewPosition(fmt.Sprintf("labeledit%d", ls.selected))
	if err != nil {
		return err
	}

	// Below bottom of frame
	if vy1 > lsy1 {
		ls.scroll += vy1 - lsy1
		return nil
	}

	// Above top of frame
	if vy0 < lsy0 {
		ls.scroll -= lsy0 - vy0
	}

	return nil
}

func (ls *labelSelect) selectPrevious(g *gocui.Gui, v *gocui.View) error {
	if ls.selected < 0 {
		return nil
	}

	ls.selected = maxInt(0, ls.selected-1)
	return ls.focusView(g)
}

func (ls *labelSelect) selectNext(g *gocui.Gui, v *gocui.View) error {
	if ls.selected < 0 {
		return nil
	}

	ls.selected = minInt(len(ls.labels)-1, ls.selected+1)
	return ls.focusView(g)
}

func (ls *labelSelect) selectItem(g *gocui.Gui, v *gocui.View) error {
	if ls.selected < 0 {
		return nil
	}

	ls.labelSelect[ls.selected] = !ls.labelSelect[ls.selected]
	return nil
}

func (ls *labelSelect) addItem(g *gocui.Gui, v *gocui.View) error {
	c := ui.inputPopup.Activate("Add a new label")

	go func() {
		input := <-c

		// Standardize label format
		input = strings.TrimSuffix(input, "\n")
		input = strings.Replace(input, " ", "-", -1)

		// Check if label already exists
		for i, label := range ls.labels {
			if input == label.String() {
				ls.labelSelect[i] = true
				ls.selected = i

				g.Update(func(gui *gocui.Gui) error {
					return ls.focusView(g)
				})

				return
			}
		}

		// Add new label, make it selected, and focus
		ls.labels = append(ls.labels, bug.Label(input))
		ls.labelSelect = append(ls.labelSelect, true)
		ls.selected = len(ls.labels) - 1

		g.Update(func(g *gocui.Gui) error {
			return nil
		})
	}()

	return nil
}

func (ls *labelSelect) abort(g *gocui.Gui, v *gocui.View) error {
	return ui.activateWindow(ui.showBug)
}

func (ls *labelSelect) saveAndReturn(g *gocui.Gui, v *gocui.View) error {
	bugLabels := ls.bug.Compile().Labels
	var selectedLabels []bug.Label
	for i, label := range ls.labels {
		if ls.labelSelect[i] {
			selectedLabels = append(selectedLabels, label)
		}
	}

	// Find the new and removed labels. This could be implemented more efficiently...
	var newLabels []string
	var rmLabels []string

	for _, selectedLabel := range selectedLabels {
		found := false
		for _, bugLabel := range bugLabels {
			if selectedLabel == bugLabel {
				found = true
			}
		}

		if !found {
			newLabels = append(newLabels, string(selectedLabel))
		}
	}

	for _, bugLabel := range bugLabels {
		found := false
		for _, selectedLabel := range selectedLabels {
			if bugLabel == selectedLabel {
				found = true
			}
		}

		if !found {
			rmLabels = append(rmLabels, string(bugLabel))
		}
	}

	if _, _, err := ls.bug.ChangeLabels(newLabels, rmLabels); err != nil {
		ui.msgPopup.Activate(msgPopupErrorTitle, err.Error())
	}

	return ui.activateWindow(ui.showBug)
}

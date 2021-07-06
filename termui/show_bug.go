package termui

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/MichaelMure/go-term-text"
	"github.com/awesome-gocui/gocui"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/util/colors"
)

const showBugView = "showBugView"
const showBugSidebarView = "showBugSidebarView"
const showBugInstructionView = "showBugInstructionView"
const showBugHeaderView = "showBugHeaderView"

const timeLayout = "Jan 2 2006"

var showBugHelp = helpBar{
	{"q", "Save and return"},
	{"←↓↑→,hjkl", "Navigation"},
	{"o", "Toggle open/close"},
	{"e", "Edit"},
	{"c", "Comment"},
	{"t", "Change title"},
}

type showBug struct {
	cache              *cache.RepoCache
	bug                *cache.BugCache
	childViews         []string
	mainSelectableView []string
	sideSelectableView []string
	selected           string
	isOnSide           bool
	scroll             int
}

func newShowBug(cache *cache.RepoCache) *showBug {
	return &showBug{
		cache: cache,
	}
}

func (sb *showBug) SetBug(bug *cache.BugCache) {
	sb.bug = bug
	sb.scroll = 0
	sb.selected = ""
	sb.isOnSide = false
}

func (sb *showBug) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	sb.childViews = nil

	v, err := g.SetView(showBugView, 0, 0, maxX*2/3, maxY-2, 0)

	if err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}

		sb.childViews = append(sb.childViews, showBugView)
		v.Frame = false
	}

	v.Clear()
	err = sb.renderMain(g, v)
	if err != nil {
		return err
	}

	v, err = g.SetView(showBugSidebarView, maxX*2/3+1, 0, maxX-1, maxY-2, 0)

	if err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}

		sb.childViews = append(sb.childViews, showBugSidebarView)
		v.Frame = false
	}

	v.Clear()
	err = sb.renderSidebar(g, v)
	if err != nil {
		return err
	}

	v, err = g.SetView(showBugInstructionView, -1, maxY-2, maxX, maxY, 0)

	if err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}

		sb.childViews = append(sb.childViews, showBugInstructionView)
		v.Frame = false
		v.FgColor = gocui.ColorWhite
	}

	v.Clear()
	_, _ = fmt.Fprint(v, showBugHelp.Render(maxX))

	_, err = g.SetViewOnTop(showBugInstructionView)
	if err != nil {
		return err
	}

	_, err = g.SetCurrentView(showBugView)
	return err
}

func (sb *showBug) keybindings(g *gocui.Gui) error {
	// Return
	if err := g.SetKeybinding(showBugView, 'q', gocui.ModNone, sb.saveAndBack); err != nil {
		return err
	}

	// Scrolling
	if err := g.SetKeybinding(showBugView, gocui.KeyPgup, gocui.ModNone,
		sb.scrollUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(showBugView, gocui.KeyPgdn, gocui.ModNone,
		sb.scrollDown); err != nil {
		return err
	}

	// Down
	if err := g.SetKeybinding(showBugView, 'j', gocui.ModNone,
		sb.selectNext); err != nil {
		return err
	}
	if err := g.SetKeybinding(showBugView, gocui.KeyArrowDown, gocui.ModNone,
		sb.selectNext); err != nil {
		return err
	}
	// Up
	if err := g.SetKeybinding(showBugView, 'k', gocui.ModNone,
		sb.selectPrevious); err != nil {
		return err
	}
	if err := g.SetKeybinding(showBugView, gocui.KeyArrowUp, gocui.ModNone,
		sb.selectPrevious); err != nil {
		return err
	}

	// Left
	if err := g.SetKeybinding(showBugView, 'h', gocui.ModNone,
		sb.left); err != nil {
		return err
	}
	if err := g.SetKeybinding(showBugView, gocui.KeyArrowLeft, gocui.ModNone,
		sb.left); err != nil {
		return err
	}
	// Right
	if err := g.SetKeybinding(showBugView, 'l', gocui.ModNone,
		sb.right); err != nil {
		return err
	}
	if err := g.SetKeybinding(showBugView, gocui.KeyArrowRight, gocui.ModNone,
		sb.right); err != nil {
		return err
	}

	// Comment
	if err := g.SetKeybinding(showBugView, 'c', gocui.ModNone,
		sb.comment); err != nil {
		return err
	}

	// Open/close
	if err := g.SetKeybinding(showBugView, 'o', gocui.ModNone,
		sb.toggleOpenClose); err != nil {
		return err
	}

	// Title
	if err := g.SetKeybinding(showBugView, 't', gocui.ModNone,
		sb.setTitle); err != nil {
		return err
	}

	// Edit
	if err := g.SetKeybinding(showBugView, 'e', gocui.ModNone,
		sb.edit); err != nil {
		return err
	}

	return nil
}

func (sb *showBug) disable(g *gocui.Gui) error {
	for _, view := range sb.childViews {
		if err := g.DeleteView(view); err != nil && !gocui.IsUnknownView(err) {
			return err
		}
	}
	return nil
}

func (sb *showBug) renderMain(g *gocui.Gui, mainView *gocui.View) error {
	maxX, _ := mainView.Size()
	x0, y0, _, _, _ := g.ViewPosition(mainView.Name())

	y0 -= sb.scroll

	snap := sb.bug.Snapshot()

	sb.mainSelectableView = nil

	createTimelineItem := snap.Timeline[0].(*bug.CreateTimelineItem)

	edited := ""
	if createTimelineItem.Edited() {
		edited = " (edited)"
	}

	bugHeader := fmt.Sprintf("[%s] %s\n\n[%s] %s opened this bug on %s%s",
		colors.Cyan(snap.Id().Human()),
		colors.Bold(snap.Title),
		colors.Yellow(snap.Status),
		colors.Magenta(snap.Author.DisplayName()),
		snap.CreateTime.Format(timeLayout),
		edited,
	)
	bugHeader, lines := text.Wrap(bugHeader, maxX, text.WrapIndent("   "))

	v, err := sb.createOpView(g, showBugHeaderView, x0, y0, maxX+1, lines, false)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprint(v, bugHeader)
	y0 += lines + 1

	for _, op := range snap.Timeline {
		viewName := op.Id().String()

		// TODO: me might skip the rendering of blocks that are outside of the view
		// but to do that we need to rework how sb.mainSelectableView is maintained

		switch op := op.(type) {

		case *bug.CreateTimelineItem:
			var content string
			var lines int

			if op.MessageIsEmpty() {
				content, lines = text.WrapLeftPadded(emptyMessagePlaceholder(), maxX-1, 4)
			} else {
				content, lines = text.WrapLeftPadded(op.Message, maxX-1, 4)
			}

			v, err := sb.createOpView(g, viewName, x0, y0, maxX+1, lines, true)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprint(v, content)
			y0 += lines + 2

		case *bug.AddCommentTimelineItem:
			edited := ""
			if op.Edited() {
				edited = " (edited)"
			}

			var message string
			if op.MessageIsEmpty() {
				message, _ = text.WrapLeftPadded(emptyMessagePlaceholder(), maxX-1, 4)
			} else {
				message, _ = text.WrapLeftPadded(op.Message, maxX-1, 4)
			}

			content := fmt.Sprintf("%s commented on %s%s\n\n%s",
				colors.Magenta(op.Author.DisplayName()),
				op.CreatedAt.Time().Format(timeLayout),
				edited,
				message,
			)
			content, lines = text.Wrap(content, maxX)

			v, err := sb.createOpView(g, viewName, x0, y0, maxX+1, lines, true)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprint(v, content)
			y0 += lines + 2

		case *bug.SetTitleTimelineItem:
			content := fmt.Sprintf("%s changed the title to %s on %s",
				colors.Magenta(op.Author.DisplayName()),
				colors.Bold(op.Title),
				op.UnixTime.Time().Format(timeLayout),
			)
			content, lines := text.Wrap(content, maxX)

			v, err := sb.createOpView(g, viewName, x0, y0, maxX+1, lines, true)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprint(v, content)
			y0 += lines + 2

		case *bug.SetStatusTimelineItem:
			content := fmt.Sprintf("%s %s the bug on %s",
				colors.Magenta(op.Author.DisplayName()),
				colors.Bold(op.Status.Action()),
				op.UnixTime.Time().Format(timeLayout),
			)
			content, lines := text.Wrap(content, maxX)

			v, err := sb.createOpView(g, viewName, x0, y0, maxX+1, lines, true)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprint(v, content)
			y0 += lines + 2

		case *bug.LabelChangeTimelineItem:
			var added []string
			for _, label := range op.Added {
				added = append(added, colors.Bold("\""+label+"\""))
			}

			var removed []string
			for _, label := range op.Removed {
				removed = append(removed, colors.Bold("\""+label+"\""))
			}

			var action bytes.Buffer

			if len(added) > 0 {
				action.WriteString("added ")
				action.WriteString(strings.Join(added, ", "))

				if len(removed) > 0 {
					action.WriteString(" and ")
				}
			}

			if len(removed) > 0 {
				action.WriteString("removed ")
				action.WriteString(strings.Join(removed, ", "))
			}

			if len(added)+len(removed) > 1 {
				action.WriteString(" labels")
			} else {
				action.WriteString(" label")
			}

			content := fmt.Sprintf("%s %s on %s",
				colors.Magenta(op.Author.DisplayName()),
				action.String(),
				op.UnixTime.Time().Format(timeLayout),
			)
			content, lines := text.Wrap(content, maxX)

			v, err := sb.createOpView(g, viewName, x0, y0, maxX+1, lines, true)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprint(v, content)
			y0 += lines + 2
		}
	}

	return nil
}

// emptyMessagePlaceholder return a formatted placeholder for an empty message
func emptyMessagePlaceholder() string {
	return colors.BlackBold(colors.WhiteBg("No description provided."))
}

func (sb *showBug) createOpView(g *gocui.Gui, name string, x0 int, y0 int, maxX int, height int, selectable bool) (*gocui.View, error) {
	v, err := g.SetView(name, x0, y0, maxX, y0+height+1, 0)

	if err != nil && !gocui.IsUnknownView(err) {
		return nil, err
	}

	sb.childViews = append(sb.childViews, name)

	if selectable {
		sb.mainSelectableView = append(sb.mainSelectableView, name)
	}

	v.Frame = sb.selected == name

	v.Clear()

	return v, nil
}

func (sb *showBug) createSideView(g *gocui.Gui, name string, x0 int, y0 int, maxX int, height int) (*gocui.View, error) {
	v, err := g.SetView(name, x0, y0, maxX, y0+height+1, 0)

	if err != nil && !gocui.IsUnknownView(err) {
		return nil, err
	}

	sb.childViews = append(sb.childViews, name)
	sb.sideSelectableView = append(sb.sideSelectableView, name)

	v.Frame = sb.selected == name

	v.Clear()

	return v, nil
}

func (sb *showBug) renderSidebar(g *gocui.Gui, sideView *gocui.View) error {
	maxX, _ := sideView.Size()
	x0, y0, _, _, _ := g.ViewPosition(sideView.Name())
	maxX += x0

	snap := sb.bug.Snapshot()

	sb.sideSelectableView = nil

	labelStr := make([]string, len(snap.Labels))
	for i, l := range snap.Labels {
		lc := l.Color()
		lc256 := lc.Term256()
		labelStr[i] = lc256.Escape() + "◼ " + lc256.Unescape() + l.String()
	}

	labels := strings.Join(labelStr, "\n")
	labels, lines := text.WrapLeftPadded(labels, maxX, 2)

	content := fmt.Sprintf("%s\n\n%s", colors.Bold("  Labels"), labels)

	v, err := sb.createSideView(g, "sideLabels", x0, y0, maxX, lines+2)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprint(v, content)

	return nil
}

func (sb *showBug) saveAndBack(g *gocui.Gui, v *gocui.View) error {
	err := sb.bug.CommitAsNeeded()
	if err != nil {
		return err
	}
	err = ui.activateWindow(ui.bugTable)
	if err != nil {
		return err
	}
	return nil
}

func (sb *showBug) scrollUp(g *gocui.Gui, v *gocui.View) error {
	mainView, err := g.View(showBugView)
	if err != nil {
		return err
	}

	_, maxY := mainView.Size()

	sb.scroll -= maxY / 2

	sb.scroll = maxInt(sb.scroll, 0)

	return nil
}

func (sb *showBug) scrollDown(g *gocui.Gui, v *gocui.View) error {
	_, maxY := v.Size()

	lastViewName := sb.mainSelectableView[len(sb.mainSelectableView)-1]

	lastView, err := g.View(lastViewName)
	if err != nil {
		return err
	}

	_, vMaxY := lastView.Size()

	_, vy0, _, _, err := g.ViewPosition(lastViewName)
	if err != nil {
		return err
	}

	maxScroll := vy0 + sb.scroll + vMaxY - maxY

	sb.scroll += maxY / 2

	sb.scroll = minInt(sb.scroll, maxScroll)

	return nil
}

func (sb *showBug) selectPrevious(g *gocui.Gui, v *gocui.View) error {
	var selectable []string
	if sb.isOnSide {
		selectable = sb.sideSelectableView
	} else {
		selectable = sb.mainSelectableView
	}

	for i, name := range selectable {
		if name == sb.selected {
			// special case to scroll up to the top
			if i == 0 {
				sb.scroll = 0
			}

			sb.selected = selectable[maxInt(i-1, 0)]
			return sb.focusView(g)
		}
	}

	if sb.selected == "" && len(selectable) > 0 {
		sb.selected = selectable[0]
	}

	return sb.focusView(g)
}

func (sb *showBug) selectNext(g *gocui.Gui, v *gocui.View) error {
	var selectable []string
	if sb.isOnSide {
		selectable = sb.sideSelectableView
	} else {
		selectable = sb.mainSelectableView
	}

	for i, name := range selectable {
		if name == sb.selected {
			sb.selected = selectable[minInt(i+1, len(selectable)-1)]
			return sb.focusView(g)
		}
	}

	if sb.selected == "" && len(selectable) > 0 {
		sb.selected = selectable[0]
	}

	return sb.focusView(g)
}

func (sb *showBug) left(g *gocui.Gui, v *gocui.View) error {
	if sb.isOnSide {
		sb.isOnSide = false
		sb.selected = ""
		return sb.selectNext(g, v)
	}

	if sb.selected == "" {
		return sb.selectNext(g, v)
	}

	return nil
}

func (sb *showBug) right(g *gocui.Gui, v *gocui.View) error {
	if !sb.isOnSide {
		sb.isOnSide = true
		sb.selected = ""
		return sb.selectNext(g, v)
	}

	if sb.selected == "" {
		return sb.selectNext(g, v)
	}

	return nil
}

func (sb *showBug) focusView(g *gocui.Gui) error {
	mainView, err := g.View(showBugView)
	if err != nil {
		return err
	}

	_, maxY := mainView.Size()

	_, vy0, _, _, err := g.ViewPosition(sb.selected)
	if err != nil {
		return err
	}

	v, err := g.View(sb.selected)
	if err != nil {
		return err
	}

	_, vMaxY := v.Size()

	vy1 := vy0 + vMaxY

	if vy0 < 0 {
		sb.scroll += vy0
		return nil
	}

	if vy1 > maxY {
		sb.scroll -= maxY - vy1
	}

	return nil
}

func (sb *showBug) comment(g *gocui.Gui, v *gocui.View) error {
	return addCommentWithEditor(sb.bug)
}

func (sb *showBug) setTitle(g *gocui.Gui, v *gocui.View) error {
	return setTitleWithEditor(sb.bug)
}

func (sb *showBug) toggleOpenClose(g *gocui.Gui, v *gocui.View) error {
	switch sb.bug.Snapshot().Status {
	case bug.OpenStatus:
		_, err := sb.bug.Close()
		return err
	case bug.ClosedStatus:
		_, err := sb.bug.Open()
		return err
	default:
		return nil
	}
}

func (sb *showBug) edit(g *gocui.Gui, v *gocui.View) error {
	snap := sb.bug.Snapshot()

	if sb.isOnSide {
		return sb.editLabels(g, snap)
	}

	if sb.selected == "" {
		return nil
	}

	op, err := snap.SearchTimelineItem(entity.CombinedId(sb.selected))
	if err != nil {
		return err
	}

	switch op := op.(type) {
	case *bug.AddCommentTimelineItem:
		return editCommentWithEditor(sb.bug, op.Id(), op.Message)
	case *bug.CreateTimelineItem:
		return editCommentWithEditor(sb.bug, op.Id(), op.Message)
	case *bug.LabelChangeTimelineItem:
		return sb.editLabels(g, snap)
	}

	ui.msgPopup.Activate(msgPopupErrorTitle, "Selected field is not editable.")
	return nil
}

func (sb *showBug) editLabels(g *gocui.Gui, snap *bug.Snapshot) error {
	ui.labelSelect.SetBug(sb.cache, sb.bug)
	return ui.activateWindow(ui.labelSelect)
}

package shell

import (
	"context"
	"fmt"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/flex"
	"github.com/vivaui/viva/ui"

	"r-git-workbench/internal/git"
	"r-git-workbench/internal/i18n"
	"r-git-workbench/internal/keymap"
	"r-git-workbench/internal/workbench"
)

var (
	tree    = ui.Tree{}
	feature = tree.Feature("shell")
	ids     = [...]ui.ID{
		tree.ID(feature, "changes"), tree.ID(feature, "log"),
		tree.ID(feature, "branches"), tree.ID(feature, "stash"),
		tree.ID(feature, "tags"), tree.ID(feature, "repositories"),
	}
	openID   = tree.ID(feature, "open")
	branchID = tree.ID(feature, "branch")
	fetchID  = tree.ID(feature, "fetch")
	pullID   = tree.ID(feature, "pull")
	pushID   = tree.ID(feature, "push")
	searchID = tree.ID(feature, "search")
)

type Model struct {
	Language     i18n.Language
	Active       keymap.ActionID
	Git          git.Installation
	DetectionErr error
	Checking     bool
	view         flex.View
	mailbox      workbench.Session
	invalidator  app.Invalidator
}

func New(language i18n.Language) *Model {
	return &Model{Language: language, Active: keymap.ShowRepositories, Checking: true}
}

// SetInitialGit applies startup detection before the window thread begins.
func (m *Model) SetInitialGit(installation git.Installation, err error) {
	m.Git, m.DetectionErr, m.Checking = installation, err, false
}

func (m *Model) Start(ctx context.Context, gitPath string) {
	m.mailbox.DetectGit(ctx, gitPath, m.invalidator.Invalidate)
}

func (m *Model) Config() app.Config {
	text := m.Language.Text
	theme := ui.LightTheme()
	config := app.Config{
		Tree: &tree, Title: text(i18n.AppName), Width: 1100, Height: 720,
		MinWidth: 720, MinHeight: 480, Invalidator: &m.invalidator,
		OnWake: func() {
			for _, result := range m.mailbox.Drain() {
				m.Git, m.DetectionErr, m.Checking = result.Installation, result.Err, false
			}
		},
	}
	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		bounds := ui.Rect{Width: size.Width, Height: size.Height}
		canvas.Fill(bounds, theme.Background)
		m.view.Draw(canvas, theme, bounds, func(f *flex.Frame) {
			f.Column().Fill().Children(func() {
				m.paintTop(f)
				f.Divider()
				f.Row().FillWidth().Grow(1).Children(func() {
					m.paintNavigation(f)
					f.Divider()
					m.paintContent(f)
				})
				f.Divider()
				m.paintStatus(f)
			})
		})
	}
	return config
}

func (m *Model) paintTop(f *flex.Frame) {
	text := m.Language.Text
	f.Row().FillWidth().Height(58).PadX(18).Gap(10).Items(flex.Center).Children(func() {
		f.Text(text(i18n.AppName)).Bold().FontSize(17)
		f.Divider().Height(22)
		f.Button(openID, text(i18n.NoRepository)).Ghost().Disabled(true)
		f.Button(branchID, text(i18n.NoBranch)).Ghost().Disabled(true)
		f.Spacer()
		f.Button(fetchID, text(i18n.Fetch)).Ghost().Disabled(true)
		f.Button(pullID, text(i18n.Pull)).Ghost().Disabled(true)
		f.Button(pushID, text(i18n.Push)).Ghost().Disabled(true)
		f.Button(searchID, text(i18n.Search)).Outline().Disabled(true)
	})
}

func (m *Model) paintNavigation(f *flex.Frame) {
	f.Column().Width(178).FillHeight().Pad(12).Gap(3).Children(func() {
		f.Text("GIT").Muted().FontSize(11).PadX(10).PadY(8)
		for i, action := range keymap.Actions {
			if f.Button(ids[i], m.Language.Text(action.Name)).Ghost().FillWidth().Clicked() {
				m.Active = action.ID
			}
		}
	})
}

func (m *Model) paintContent(f *flex.Frame) {
	text := m.Language.Text
	f.Column().Fill().Pad(28).Gap(14).Children(func() {
		f.Text(text(keymap.Actions[m.Active].Name)).FontSize(22).Bold()
		f.Divider()
		f.Spacer()
		f.Row().FillWidth().Justify(flex.Center).Children(func() {
			f.Text(text(i18n.SelectRepository)).Muted()
		})
		f.Spacer()
	})
}

func (m *Model) paintStatus(f *flex.Frame) {
	text := m.Language.Text
	f.Row().FillWidth().Height(36).PadX(16).Gap(12).Items(flex.Center).Children(func() {
		f.Text(text(i18n.NoRepository)).Muted().FontSize(12)
		f.Spacer()
		switch {
		case m.Checking:
			f.Text(text(i18n.CheckingGit)).Muted().FontSize(12)
		case m.DetectionErr != nil:
			f.Text(text(i18n.GitUnavailable)).Color(ui.RGBA(160, 49, 46, 1)).FontSize(12)
		case !m.Git.Version.Capabilities().CanWrite:
			f.Text(text(i18n.GitUnsupported)).Color(ui.RGBA(160, 49, 46, 1)).FontSize(12)
		default:
			f.Text(text(i18n.Ready)).Muted().FontSize(12)
		}
		if m.Git.Path != "" {
			f.Text(fmt.Sprintf("%s %s", text(i18n.GitVersion), m.Git.Version.Raw)).Mono().FontSize(12)
		}
	})
}

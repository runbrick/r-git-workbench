package main

import (
	"context"
	"flag"
	"log"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"

	"r-git-workbench/internal/git"
	"r-git-workbench/internal/i18n"
	"r-git-workbench/internal/ui/shell"
)

func main() {
	gitPath := flag.String("git", "", "Git executable path")
	flag.Parse()
	m := shell.New(i18n.FromPreferred(app.PreferredLanguages()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	installation, err := git.Detect(ctx, *gitPath)
	m.SetInitialGit(installation, err)
	var canvas ui.Canvas
	if err := app.RunCanvas(m.Config(), &canvas); err != nil {
		log.Fatal(err)
	}
}

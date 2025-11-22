package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2/app"
	"github.com/pgsdf/pgpart/internal/partition"
	"github.com/pgsdf/pgpart/internal/ui"
)

func main() {
	fmt.Println("PGPart - Partition Manager for FreeBSD/GhostBSD")
	fmt.Println("================================================")

	if err := partition.CheckPrivileges(); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: %v\n", err)
		fmt.Println("Some operations may be restricted. Run with sudo for full functionality.")
	}

	application := app.New()
	application.Settings().SetTheme(&CustomTheme{})

	mainWindow := ui.NewMainWindow(application)
	mainWindow.Show()
}

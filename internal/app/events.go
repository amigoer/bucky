package app

import (
	"github.com/amigoer/bucky/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func init() {
	application.RegisterEvent[[]string]("files:dropped")
	application.RegisterEvent[core.TransferTask]("transfer:updated")
	application.RegisterEvent[UsageProgress]("usage:progress")
}

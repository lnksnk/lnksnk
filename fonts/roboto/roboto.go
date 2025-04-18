package roboto

import (
	"embed"
	_ "embed"
)

//go:embed *
var RobotoFS embed.FS

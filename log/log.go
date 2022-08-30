package log

import "github.com/fatih/color"

var (
	Error   = color.New(color.FgRed)
	Success = color.New(color.FgGreen)
	Info    = color.New(color.FgBlue)
)

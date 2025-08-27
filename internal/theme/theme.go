package theme

import "fmt"

type Theme struct {
    OK, Info, Warn, Err, Dim, Accent, Reset string
}

func FromName(name string) Theme {
    const (
        reset = "[0m"
        bold = "[1m"
        dim = "[2m"
        red = "[31m"
        green = "[32m"
        yellow = "[33m"
        blue = "[34m"
    )
    switch name {
    case "mono":
        return Theme{}
    default:
        return Theme{OK: green, Info: blue, Warn: yellow, Err: red, Dim: dim, Accent: bold, Reset: reset}
    }
}

func Sprintf(t Theme, color, format string, a ...any) string {
    if color == "" { return fmt.Sprintf(format, a...) }
    return color + fmt.Sprintf(format, a...) + t.Reset
}
package pretty

import "fmt"

type PrettyPrintOption func(p *prettyPrinter)

type PrettyPrinter interface {
	Print(s string)
	Printf(format string, a ...any)
	PrintWarning(s string)
	PrintWarningf(format string, a ...any)
	PrintError(s string)
	PrintErrorf(format string, a ...any)
}

type prettyPrinter struct {
	InfoColor int32 `json:"infoColor"`
	WarnColor int32 `json:"warnColor"`
	ErrColor  int32 `json:"errorColor"`
}

func WithInfoColor(c int32) PrettyPrintOption {
	return func(p *prettyPrinter) {
		p.InfoColor = c
	}
}

func WithWarnColor(c int32) PrettyPrintOption {
	return func(p *prettyPrinter) {
		p.WarnColor = c
	}
}

func WithErrColor(c int32) PrettyPrintOption {
	return func(p *prettyPrinter) {
		p.ErrColor = c
	}
}

func NewPrettyPrinter(opts ...PrettyPrintOption) *prettyPrinter {
	const (
		infoColor = int32(92)
		warnColor = int32(93)
		errColor  = int32(91)
	)

	printer := &prettyPrinter{
		InfoColor: infoColor,
		WarnColor: warnColor,
		ErrColor:  errColor,
	}
	for _, opt := range opts {
		opt(printer)
	}
	return printer
}

func (p *prettyPrinter) Print(s ...any) {
	fmt.Printf("\x1b[1;%dm%s\x1b[0m\n", p.InfoColor, s)
}

func (p *prettyPrinter) Printf(format string, a ...any) {
	fstring := fmt.Sprintf(format, a)
	p.Print(fstring)
}

func (p *prettyPrinter) PrintWarning(s ...any) {
	fmt.Printf("\x1b[1;%dm%s\x1b[0m\n", p.WarnColor, s)
}

func (p *prettyPrinter) PrintWarningf(format string, a ...any) {
	fstring := fmt.Sprintf(format, a)
	p.PrintWarning(fstring)
}

func (p *prettyPrinter) PrintError(s ...any) {
	fmt.Printf("\x1b[1;%dm%s\x1b[0m\n", p.ErrColor, s)
}

func (p *prettyPrinter) PrintErrorf(format string, a ...any) {
	fstring := fmt.Sprintf(format, a)
	p.PrintError(fstring)
}

func Print(s ...any) {
	const (
		infoColor = int32(92)
	)
	fmt.Printf("\x1b[1;%dm%s\x1b[0m\n", infoColor, s)
}

func Printf(format string, a ...any) {
	const (
		infoColor = int32(92)
	)
	fstring := fmt.Sprintf(format, a)
	fmt.Printf("\x1b[1;%dm%s\x1b[0m\n", infoColor, fstring)
}

func PrintWarning(s ...any) {
	const (
		warnColor = int32(93)
	)
	fmt.Printf("\x1b[1;%dm%s\x1b[0m\n", warnColor, s)
}

func PrintWarningf(format string, a ...any) {
	const (
		warnColor = int32(93)
	)
	fstring := fmt.Sprintf(format, a)
	fmt.Printf("\x1b[1;%dm%s\x1b[0m\n", warnColor, fstring)
}

func PrintError(s ...any) {
	const (
		errColor = int32(91)
	)
	fmt.Printf("\x1b[1;%dm%s\x1b[0m\n", errColor, s)
}

func PrintErrorf(format string, a ...any) {
	const (
		errColor = int32(91)
	)
	fstring := fmt.Sprintf(format, a)
	fmt.Printf("\x1b[1;%dm%s\x1b[0m\n", errColor, fstring)
}

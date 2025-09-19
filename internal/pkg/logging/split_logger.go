package logging

import (
    "io"
    "os"

    glog "github.com/labstack/gommon/log"
)

// SplitLogger implements echo.Logger behavior using two gommon loggers:
// - out: writes Debug/Info/Warn/Print to stdout (or a provided writer)
// - err: writes Error/Fatal/Panic to stderr (or a provided writer)
// It satisfies the echo.Logger interface signature without importing echo here.
type SplitLogger struct {
    out   *glog.Logger
    err   *glog.Logger
}

// NewSplitLogger creates a SplitLogger with stdout and stderr as defaults.
func NewSplitLogger() *SplitLogger {
    lout := glog.New("")
    lout.SetOutput(os.Stdout)
    lout.SetLevel(glog.INFO)

    lerr := glog.New("")
    lerr.SetOutput(os.Stderr)
    lerr.SetLevel(glog.INFO)

    return &SplitLogger{out: lout, err: lerr}
}

// Output returns the non-error output writer.
func (l *SplitLogger) Output() io.Writer { return l.out.Output() }

// SetOutput sets both writers to the same output.
// Note: this removes the split; used only if explicitly called by host.
func (l *SplitLogger) SetOutput(w io.Writer) {
    l.out.SetOutput(w)
    l.err.SetOutput(w)
}

func (l *SplitLogger) Prefix() string { return l.out.Prefix() }
func (l *SplitLogger) SetPrefix(p string) {
    l.out.SetPrefix(p)
    l.err.SetPrefix(p)
}

func (l *SplitLogger) Level() glog.Lvl { return l.out.Level() }
func (l *SplitLogger) SetLevel(v glog.Lvl) {
    l.out.SetLevel(v)
    l.err.SetLevel(v)
}

func (l *SplitLogger) SetHeader(h string) {
    l.out.SetHeader(h)
    l.err.SetHeader(h)
}

func (l *SplitLogger) Print(i ...interface{})                 { l.out.Print(i...) }
func (l *SplitLogger) Printf(format string, args ...interface{}) { l.out.Printf(format, args...) }
func (l *SplitLogger) Printj(j glog.JSON)                      { l.out.Printj(j) }

func (l *SplitLogger) Debug(i ...interface{})                  { l.out.Debug(i...) }
func (l *SplitLogger) Debugf(format string, args ...interface{}) { l.out.Debugf(format, args...) }
func (l *SplitLogger) Debugj(j glog.JSON)                     { l.out.Debugj(j) }

func (l *SplitLogger) Info(i ...interface{})                   { l.out.Info(i...) }
func (l *SplitLogger) Infof(format string, args ...interface{})  { l.out.Infof(format, args...) }
func (l *SplitLogger) Infoj(j glog.JSON)                      { l.out.Infoj(j) }

func (l *SplitLogger) Warn(i ...interface{})                   { l.out.Warn(i...) }
func (l *SplitLogger) Warnf(format string, args ...interface{})  { l.out.Warnf(format, args...) }
func (l *SplitLogger) Warnj(j glog.JSON)                      { l.out.Warnj(j) }

func (l *SplitLogger) Error(i ...interface{})                  { l.err.Error(i...) }
func (l *SplitLogger) Errorf(format string, args ...interface{}) { l.err.Errorf(format, args...) }
func (l *SplitLogger) Errorj(j glog.JSON)                     { l.err.Errorj(j) }

func (l *SplitLogger) Fatal(i ...interface{})                  { l.err.Fatal(i...) }
func (l *SplitLogger) Fatalj(j glog.JSON)                     { l.err.Fatalj(j) }
func (l *SplitLogger) Fatalf(format string, args ...interface{}) { l.err.Fatalf(format, args...) }

func (l *SplitLogger) Panic(i ...interface{})                  { l.err.Panic(i...) }
func (l *SplitLogger) Panicj(j glog.JSON)                     { l.err.Panicj(j) }
func (l *SplitLogger) Panicf(format string, args ...interface{}) { l.err.Panicf(format, args...) }


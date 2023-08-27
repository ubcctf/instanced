package main

import (
	"fmt"
	"io"
	"os"

	"github.com/labstack/gommon/log"
	"github.com/rs/zerolog"
)

type EchoLog struct {
	log zerolog.Logger
}

func NewEchoLog(l zerolog.Logger) EchoLog {
	return EchoLog{
		log: l.With().Str("component", "echo").Logger(),
	}
}

func (l EchoLog) Output() io.Writer {
	// Not Implemented
	return os.Stderr
}

func (l EchoLog) SetOutput(w io.Writer) {
	// Not Implemented
}

func (l EchoLog) Prefix() string {
	return "echo"
}

func (l EchoLog) SetPrefix(p string) {
	// Not Implemented
}

func (l EchoLog) Level() log.Lvl {
	return zltoel(l.log.GetLevel())
}

func (l EchoLog) SetLevel(v log.Lvl) {
	// Not Implemented
}

func (l EchoLog) SetHeader(h string) {
	// Not Implemented
}

func (l EchoLog) Print(i ...interface{}) {
	l.log.Log().Msg(fmt.Sprint(i...))
}

func (l EchoLog) Printf(format string, args ...interface{}) {
	l.log.Log().Msgf(format, args...)
}

func (l EchoLog) Printj(j log.JSON) {
	l.log.Log().Any("json", j).Msg("json")
}

func (l EchoLog) Debug(i ...interface{}) {
	l.log.Debug().Msg(fmt.Sprint(i...))
}

func (l EchoLog) Debugf(format string, args ...interface{}) {
	l.log.Debug().Msgf(format, args...)
}

func (l EchoLog) Debugj(j log.JSON) {
	l.log.Debug().Any("json", j).Msg("json")
}

func (l EchoLog) Info(i ...interface{}) {
	l.log.Info().Msg(fmt.Sprint(i...))
}

func (l EchoLog) Infof(format string, args ...interface{}) {
	l.log.Info().Msgf(format, args...)
}

func (l EchoLog) Infoj(j log.JSON) {
	l.log.Info().Any("json", j).Msg("json")
}

func (l EchoLog) Warn(i ...interface{}) {
	l.log.Warn().Msg(fmt.Sprint(i...))
}

func (l EchoLog) Warnf(format string, args ...interface{}) {
	l.log.Warn().Msgf(format, args...)
}

func (l EchoLog) Warnj(j log.JSON) {
	l.log.Warn().Any("json", j).Msg("json")
}

func (l EchoLog) Error(i ...interface{}) {
	l.log.Error().Msg(fmt.Sprint(i...))
}

func (l EchoLog) Errorf(format string, args ...interface{}) {
	l.log.Error().Msgf(format, args...)
}

func (l EchoLog) Errorj(j log.JSON) {
	l.log.Error().Any("json", j).Msg("json")
}

func (l EchoLog) Fatal(i ...interface{}) {
	l.log.Fatal().Msg(fmt.Sprint(i...))
}

func (l EchoLog) Fatalj(j log.JSON) {
	l.log.Fatal().Any("json", j).Msg("json")
}

func (l EchoLog) Fatalf(format string, args ...interface{}) {
	l.log.Fatal().Msgf(format, args...)
}

func (l EchoLog) Panic(i ...interface{}) {
	l.log.Panic().Msg(fmt.Sprint(i...))
}

func (l EchoLog) Panicj(j log.JSON) {
	l.log.Panic().Any("json", j).Msg("json")
}

func (l EchoLog) Panicf(format string, args ...interface{}) {
	l.log.Panic().Msgf(format, args...)
}

func zltoel(l zerolog.Level) log.Lvl {
	switch l {
	case zerolog.Disabled:
		return log.OFF
	case zerolog.NoLevel:
		return log.OFF
	case zerolog.PanicLevel:
		return log.ERROR
	case zerolog.FatalLevel:
		return log.ERROR
	case zerolog.ErrorLevel:
		return log.ERROR
	case zerolog.WarnLevel:
		return log.WARN
	case zerolog.InfoLevel:
		return log.INFO
	case zerolog.DebugLevel:
		return log.DEBUG
	case zerolog.TraceLevel:
		return log.DEBUG
	}
	return log.OFF
}

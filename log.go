package main

import (
	"bytes"
	"io"
	"log"
	"os"
)

// logger is the logger struct
type logger struct {
	Debug *log.Logger
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
	Fatal *log.Logger
}

// Log is the global logger
var Log logger

var logBuf bytes.Buffer
var debugBuf bytes.Buffer
var infoBuf bytes.Buffer
var warnBuf bytes.Buffer
var errorBuf bytes.Buffer
var fatalBuf bytes.Buffer

// InitLogger must be called once to initialize the global logger
func InitLogger() {
	mw := io.MultiWriter(os.Stdout, &logBuf)

	Log.Debug = log.New(io.MultiWriter(mw, &debugBuf), "DEBUG ", log.LstdFlags|log.Lmsgprefix)
	Log.Info = log.New(io.MultiWriter(mw, &infoBuf), " INFO ", log.LstdFlags|log.Lmsgprefix)
	Log.Warn = log.New(io.MultiWriter(mw, &warnBuf), " WARN ", log.LstdFlags|log.Lmsgprefix)
	Log.Error = log.New(io.MultiWriter(mw, &warnBuf), "ERROR ", log.LstdFlags|log.Lmsgprefix)
	Log.Fatal = log.New(io.MultiWriter(mw, &fatalBuf), "FATAL ", log.LstdFlags|log.Lmsgprefix)
}

func (_log *logger) String() string {
	// Reading the log as string will set its offset - to prepare for "someone else"
	// reading it, we reset it and write the string again
	logContent := logBuf.String()
	logBuf.Reset()
	logBuf.WriteString(logContent)

	return logContent
}

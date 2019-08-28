package util

import (
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
)

var (
	Log *logrus.Logger
)

func init(){
	Log = NewLogger()
}

func NewLogger() *logrus.Logger {
	if Log != nil {
		return Log
	}
	pathMap := lfshook.PathMap{
		logrus.InfoLevel:  "../log/info.log",
		logrus.ErrorLevel: "../log/error.log",
	}
	Log = logrus.New()
	Log.Hooks.Add(lfshook.NewHook(
		pathMap,
		&logrus.JSONFormatter{},
	))
	return Log
}
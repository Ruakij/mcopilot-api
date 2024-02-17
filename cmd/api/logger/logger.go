package logger

import (
	"log"
	"os"
)

var Info log.Logger = *log.New(os.Stdout, "[INFO]\t", log.Ltime|log.Lshortfile)
var Warn log.Logger = *log.New(os.Stderr, "[WARN]\t", log.Ltime|log.Lshortfile)
var Error log.Logger = *log.New(os.Stderr, "[ERROR]\t", log.Ltime|log.Lshortfile)

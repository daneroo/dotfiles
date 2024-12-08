package logsetup

import (
	"fmt"
	"log"
	"time"
)

const (
	fmtRFC3339Millis = "2006-01-02T15:04:05.000Z07:00"
)

type logWriter struct {
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().UTC().Format(fmtRFC3339Millis) + " - " + string(bytes))
}

// SetupFormat initializes logging
func SetupFormat() {
	log.SetFlags(0)
	log.SetOutput(new(logWriter))
}

package main

import (
	"fmt"

	lm "github.com/hrfee/jfa-go/logmessages"
)

func (app *appContext) HardRestart() error {
	return fmt.Errorf(lm.FailedHardRestartWindows)
}

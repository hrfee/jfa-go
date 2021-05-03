package main

import "fmt"

func (app *appContext) HardRestart() error {
	return fmt.Errorf("hard restarts not available on windows")
}

//go:build !tray
// +build !tray

package main

var TRAY = false

func BuildTagsTray() {}

func RunTray() {}

func QuitTray() {}

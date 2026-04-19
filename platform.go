package main

import "runtime"

func isMacOS() bool   { return runtime.GOOS == "darwin" }
func isLinux() bool   { return runtime.GOOS == "linux" }
func isWindows() bool { return runtime.GOOS == "windows" }

package main

import (
	"fmt"
	"log"
	"time"
)

const (
	cReset  = "\033[0m"
	cGreen  = "\033[0;32m"
	cYellow = "\033[1;33m"
	cRed    = "\033[0;31m"
	cCyan   = "\033[0;36m"
	cDim    = "\033[2m"
)

func printOK(s string)   { fmt.Printf("%s  ✔  %s%s\n", cGreen, s, cReset) }
func printInfo(s string) { fmt.Printf("%s  →  %s%s\n", cYellow, s, cReset) }
func printWarn(s string) { fmt.Printf("%s  ⚠  %s%s\n", cYellow, s, cReset) }
func printFail(s string) { fmt.Printf("%s  ✘  %s%s\n", cRed, s, cReset) }

func logf(format string, args ...any) {
	log.Printf(format, args...)
}

func debugf(format string, args ...any) {
	if *flagDebug {
		log.Printf("[debug] "+format, args...)
	}
}

func ts() string {
	return time.Now().Format("15:04:05")
}

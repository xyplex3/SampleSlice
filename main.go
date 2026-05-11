// Command sampleslice detects transients in a WAV file and slices them
// into Akai MPC programs (.xpj/.xpm) or Kurzweil KRZ sample libraries.
package main

import (
	"os"

	"sampleslice/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
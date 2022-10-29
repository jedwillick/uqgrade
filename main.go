package main

import (
	"fmt"
	"path"
	"runtime"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	var debug bool
	log.SetFormatter(&log.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
		},
	})

	var rootCmd = &cobra.Command{
		Use:                   "uqgrade CODES...",
		Short:                 "UQGrade is a command line tool to calculate your UQ grade",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			if debug {
				log.SetLevel(log.DebugLevel)
				log.SetReportCaller(true)
			}

			courses, invalid := scrap(args)
			if len(invalid) > 0 {
				log.Warn("Invalid courses codes: ", invalid)
			}

			tui(courses)
		},
	}
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Debug mode")
	rootCmd.Execute()
}

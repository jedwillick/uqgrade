package main

import (
	"fmt"
	"path"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func currentSemester(when *When) {
	now := time.Now()
	month := now.Month()
	when.Year = now.Year()
	if month >= 3 && month <= 7 {
		when.Semester = 1
	} else if month >= 8 && month <= 11 {
		when.Semester = 2
	} else {
		when.Semester = 3
		if month <= 2 {
			when.Year = when.Year - 1
		}
	}
	log.Debug("Current semester: ", when)
}

func fullyQualifiedWhen(when *When) error {
	if when.Semester < 0 || when.Semester > 3 {
		return fmt.Errorf("invalid semester: %d", when.Semester)
	}
	if when.Year < 0 {
		return fmt.Errorf("invalid year: %d", when.Year)
	}
	if when.Year == 0 {
		when.Year = time.Now().Year()
	}
	switch when.Semester {
	case 0:
		currentSemester(when)
		fullyQualifiedWhen(when)
	case 1, 2:
		when.FullyQualified = fmt.Sprintf("Semester %d, %d", when.Semester, when.Year)
	case 3:
		when.FullyQualified = fmt.Sprintf("Summer Semester, %d", when.Year)
	}
	log.Debug("Fully qualified when: ", when)
	return nil
}

func main() {
	var debug bool
	var when When
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
			err := fullyQualifiedWhen(&when)
			if err != nil {
				log.Fatal(err)
			}
			courses, invalid := scrap(args, when)
			if len(invalid) > 0 {
				log.Warn("Invalid courses codes: ", invalid)
			}

			tui(courses, when)
		},
	}
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Debug mode")
	rootCmd.Flags().IntVarP(&when.Semester, "semester", "s", 0, "Semester (0 = current, 1 = Sem 1, 2 = Sem 2, 3 = Summer)")
	rootCmd.Flags().IntVarP(&when.Year, "year", "y", 0, "Year (0 = current)")
	rootCmd.Execute()
}

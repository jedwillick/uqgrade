package main

import (
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/gocolly/colly/v2"
)

type Assessment struct {
	Name   string
	Weight float64
}

type Course struct {
	Name       string
	Assessment []Assessment
}

func scrap(codes []string) ([]Course, []string) {
	// Instantiate default collector
	c := colly.NewCollector(
		colly.AllowedDomains("my.uq.edu.au", "course-profiles.uq.edu.au"),
	)

	courses := make([]Course, 0, len(codes))
	var current string

	c.OnXML("(//table[@class='offerings']//a[contains(.,'Semester 2, 2022')]/../..//a[@class='profile-available'])[1]", func(f *colly.XMLElement) {
		link := f.Attr("href")
		if link == "" {
			log.Debugf("%s: no link found\n", current)
			return
		}
		c.Visit(strings.Replace(link, "section_1", "section_5", 1))
	})

	c.OnHTML("div.columns table tbody", func(e *colly.HTMLElement) {
		assessment := []Assessment{}

		e.ForEach("tr", func(_ int, e *colly.HTMLElement) {
			tokens := strings.Split(e.ChildText("td:nth-child(1) > a"), "\n")

			name := strings.TrimSpace(tokens[len(tokens)-1])
			rawWeight := e.ChildText("td:nth-child(3)")
			var re = regexp.MustCompile(`\d+`)
			weight, err := strconv.ParseFloat(re.FindString(rawWeight), 64)
			if err != nil {
				log.Debugf("%s:%s: %s\n", current, name, err.Error())
				return
			}
			assessment = append(assessment, Assessment{name, weight})
		})
		if len(assessment) == 0 {
			return
		}
		course := Course{current, assessment}
		log.Debug(course)
		courses = append(courses, course)
	})

	c.OnResponse(func(r *colly.Response) {
		log.Debug("Response Code:", r.StatusCode)
	})

	c.OnRequest(func(r *colly.Request) {
		log.Debug("Request:", r.URL.String())
	})

	var invalidCodes []string
	for _, course := range codes {
		current = strings.ToUpper(course)
		startLen := len(courses)
		c.Visit("https://my.uq.edu.au/programs-courses/course.html?course_code=" + current)
		if len(courses) == startLen {
			invalidCodes = append(invalidCodes, current)
		}
	}
	log.Debug("FOUND: ", courses)
	log.Debug("INVALID: ", invalidCodes)
	return courses, invalidCodes
}

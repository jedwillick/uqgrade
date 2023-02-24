package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/gocolly/colly/v2"
)

type Assessment struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

type Course struct {
	Name       string       `json:"name"`
	Assessment []Assessment `json:"assessment"`
}

type When struct {
	Semester       int
	Year           int
	FullyQualified string
}

func scrap(codes []string, when When) ([]Course, []string) {
	// Instantiate default collector
	c := colly.NewCollector(
		colly.AllowedDomains("my.uq.edu.au", "course-profiles.uq.edu.au"),
	)

	courses := make([]Course, 0, len(codes))
	var current string

	c.OnXML(fmt.Sprintf("(//table[@class='offerings']//a[contains(.,'%s')]/../..//a[@class='profile-available'])[1]", when.FullyQualified), func(f *colly.XMLElement) {
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
	for _, code := range codes {
		cache := path.Join(CACHE, fmt.Sprintf("%d-%d-%s", when.Year, when.Semester, code))
		current = strings.ToUpper(code)
		startLen := len(courses)
		raw, err := os.ReadFile(cache)
		if err == nil {
			log.Debugf("CACHE: found %s @ %s", code, cache)
			var course Course
			err = json.Unmarshal(raw, &course)
			if err == nil {
				courses = append(courses, course)
				continue
			}
		}

		err = c.Visit("https://my.uq.edu.au/programs-courses/course.html?course_code=" + current)
		if err != nil || len(courses) == startLen {
			invalidCodes = append(invalidCodes, current)
		} else {
			log.Debugf("CACHE: caching %s @ %s", code, cache)
			data, err := json.Marshal(courses[len(courses)-1])
			if err != nil {
				log.Errorf("CACHE: unable to cache %s: %s", cache, err)
				continue
			}
			err = os.WriteFile(cache, data, 0644)
			if err != nil {
				log.Errorf("CACHE: unable to cache %s: %s", cache, err)
				os.Remove(cache) // Avoid partially written cache files
				continue
			}

		}
	}
	log.Debug("FOUND: ", courses)
	log.Debug("INVALID: ", invalidCodes)
	return courses, invalidCodes
}

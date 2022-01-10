package scrape

import (
	"github.com/gocolly/colly/v2"
)

type Result struct {
	Title       *string
	Description *string
}

func Scrape(href string) (Result, error) {
	r := Result{}
	c := colly.NewCollector()

	c.OnHTML(`title`, func(e *colly.HTMLElement) {
		if e == nil {
			return
		}
		title := e.Text
		if title != "" {
			r.Title = &title
		}
	})

	c.OnHTML(`meta[name=description]`, func(e *colly.HTMLElement) {
		if e == nil {
			return
		}
		content := e.Attr("content")
		if content != "" {
			r.Description = &content
		}
	})

	if err := c.Visit(href); err != nil {
		return r, err
	}

	return r, nil
}

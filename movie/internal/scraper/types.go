package scraper

import "fmt"

type Movie struct {
	Title  string
	Year   int
	Score  int
	Source string
}

func movieKey(title string, year int) string {
	if year > 0 {
		return fmt.Sprintf("%s|%d", title, year)
	}
	return title
}

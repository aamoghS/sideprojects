package agent

import "fmt"

func FormatTitle(title string, year int) string {
	if year > 0 {
		return fmt.Sprintf("%s (%d)", title, year)
	}
	return title
}

func PickToMovie(p Pick, source string) Movie {
	return Movie{
		Title:  p.Title,
		Year:   p.Year,
		Plot:   p.Plot,
		Source: source,
	}
}

func MovieKey(title string, year int) string {
	if year > 0 {
		return fmt.Sprintf("%s|%d", title, year)
	}
	return title
}

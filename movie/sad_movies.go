package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SearchResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				Id    string `json:"id"`
				Title string `json:"title"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type CommentsResponse []struct {
	Data struct {
		Children []struct {
			Data struct {
				Body  string `json:"body"`
				Score int    `json:"score"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func fetchPlot(client *http.Client, title string) string {
	// url encode the title and append "film" to help Wikipedia find movies specifically
	encodedTitle := url.QueryEscape(title + " film")
	wikiURL := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=opensearch&search=%s&limit=1&format=json", encodedTitle)
	
	req, err := http.NewRequest("GET", wikiURL, nil)
	if err != nil {
		return "Plot unavailable."
	}
	req.Header.Set("User-Agent", "golang:sad-movies-finder:v1.0 (by /u/anonymous)")
	
	resp, err := client.Do(req)
	if err != nil {
		return "Plot unavailable."
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "Plot unavailable."
	}

	var result []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "Plot unavailable."
	}

	// Wikipedia opensearch returns: [searchTerm, [titles], [descriptions], [urls]]
	if len(result) >= 3 {
		descriptions, ok := result[2].([]interface{})
		if ok && 0 < len(descriptions) {
			descStr, okStr := descriptions[0].(string)
			// Sometimes Wikipedia just returns the title again or empty if it's a disambiguation
			if okStr && descStr != "" && !strings.Contains(descStr, "may refer to:") {
				return descStr
			}
		}
	}
	return "Plot unavailable."
}

func main() {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}
	
	fmt.Println("Searching Reddit (r/MovieSuggestions) for niche/underrated sad movie threads...")
	searchURL := "https://www.reddit.com/r/MovieSuggestions/search.json?q=niche+sad+movie+OR+underrated+sad+movie&restrict_sr=on&sort=comments&t=all&limit=1"

	fmt.Println()
	
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("User-Agent", "golang:sad-movies-finder:v1.0 (by /u/anonymous)")
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error fetching search results:", err)
		return
	}
	defer resp.Body.Close()
	
	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}
	
	if len(searchResp.Data.Children) == 0 {
		fmt.Println("Could not find any relevant threads. The API might have rate limited the request.")
		return
	}
	
	threadID := searchResp.Data.Children[0].Data.Id
	threadTitle := searchResp.Data.Children[0].Data.Title
	
	fmt.Printf("Found popular thread: \"%s\"\n\n", threadTitle)
	fmt.Println("Fetching top movie suggestions and their plots (this might take a moment)...")
	
	// Fetching up to 30 comments to ensure we can get 10 good movies
	commentsURL := fmt.Sprintf("https://www.reddit.com/r/MovieSuggestions/comments/%s.json?sort=top&limit=30", threadID)
	req2, _ := http.NewRequest("GET", commentsURL, nil)
	req2.Header.Set("User-Agent", "golang:sad-movies-finder:v1.0 (by /u/anonymous)")
	
	resp2, err := client.Do(req2)
	if err != nil {
		fmt.Println("Error fetching comments:", err)
		return
	}
	defer resp2.Body.Close()
	
	var commentsResp CommentsResponse
	if err := json.NewDecoder(resp2.Body).Decode(&commentsResp); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}
	
	if len(commentsResp) < 2 {
		fmt.Println("Could not parse comments.")
		return
	}
	
	fmt.Println("\nTop Niche Sad Movies Suggested by Reddit:")
	fmt.Println(strings.Repeat("-", 80))
	
	count := 0
	for _, child := range commentsResp[1].Data.Children {
		body := strings.TrimSpace(child.Data.Body)
		score := child.Data.Score
		
		if body == "" || body == "[deleted]" || body == "[removed]" || score <= 1 {
			continue // Skip bad or low-voted entries
		}
		
		// Grab the title from the first bit of the comment
		movieSuggestion := strings.TrimSpace(strings.SplitN(body, "\n", 2)[0])
		
		// Ensure it's not a generic conversational start
		if len(movieSuggestion) > 55 {
			movieSuggestion = movieSuggestion[:52] + "..."
		}
		
		if movieSuggestion != "" {
			count++
			plot := fetchPlot(client, movieSuggestion) // Grab the plot from Wikipedia
			fmt.Printf("%2d. %s\n    (Upvotes: %d)\n    Plot: %s\n\n", count, movieSuggestion, score, plot)
			if count >= 10 { // Limit to top 10 to avoid too many API calls
				break
			}
		}
	}
}

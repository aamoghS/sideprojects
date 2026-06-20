package agent

type Pick struct {
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Plot     string `json:"plot"`
	WikiPage string `json:"wiki_page,omitempty"`
}

type Agent struct {
	Name       string   `json:"name"`
	ID         string   `json:"id"`
	UserAgent  string   `json:"user_agent"`
	Proxy      string   `json:"proxy,omitempty"`
	Subreddits []string `json:"subreddits"`
	Queries    []string `json:"queries"`
	Limit      int      `json:"limit"`
	Picks      []Pick   `json:"picks"`
}

type Config struct {
	Proxy   string   `json:"proxy,omitempty"`
	Proxies []string `json:"proxies,omitempty"`
	Agents  []Agent  `json:"agents"`
}

type Movie struct {
	Title  string
	Year   int
	Plot   string
	Score  int
	Source string
}

type Result struct {
	Agent  Agent
	Movies []Movie
	Error  string
	Proxy  string
}

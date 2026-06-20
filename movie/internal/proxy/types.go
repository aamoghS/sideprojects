package proxy

type Entry struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	Enabled  bool     `json:"enabled"`
	Provider string   `json:"provider,omitempty"`
	Notes    string   `json:"notes,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

type Docket struct {
	Rotation     string  `json:"rotation"`
	DefaultProxy string  `json:"default_proxy,omitempty"`
	Proxies      []Entry `json:"proxies"`
}

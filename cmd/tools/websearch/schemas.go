// Package websearch handles the mcp for searching the web.
// The mcp is defined in the python scripts; this script serves as the schema
// for the Tavily mcp client, as they provide their own mcp server.
package websearch

type SearchRequest struct {
	Query             string   `json:"query"`
	SearchDepth       string   `json:"search_depth,omitempty"`
	Topic             string   `json:"topic,omitempty"`
	TimeRange         string   `json:"time_range,omitempty"`
	MaxResults        int      `json:"max_results,omitempty"`
	IncludeImages     bool     `json:"include_images,omitempty"`
	IncludeAnswer     bool     `json:"include_answer,omitempty"`
	IncludeRawContent bool     `json:"include_raw_content,omitempty"`
	IncludeDomains    []string `json:"include_domains,omitempty"`
	ExcludeDomains    []string `json:"exclude_domains,omitempty"`
	Country           string   `json:"country,omitempty"`
}

type ImageResult struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type SearchResult struct {
	Title         string        `json:"title"`
	URL           string        `json:"url"`
	Content       string        `json:"content"`
	Score         float64       `json:"score"`
	RawContent    *string       `json:"raw_content"`
	PublishedDate *string       `json:"published_date"`
	Favicon       *string       `json:"favicon"`
	Images        []ImageResult `json:"images"`
}

type SearchResponse struct {
	Query        string         `json:"query"`
	Answer       string         `json:"answer"`
	Results      []SearchResult `json:"results"`
	Images       []ImageResult  `json:"images"`
	ResponseTime float64        `json:"response_time"`
	RequestID    string         `json:"request_id"`
}

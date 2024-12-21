package model

type Resource struct {
	ID            int    `json:"id"`
	Count         int    `json:"count,omitempty"`
	URL           string `json:"url,omitempty"`
	Type          string `json:"type"`
	PathToStorage string `json:"path_to_storage,omitempty"`
}

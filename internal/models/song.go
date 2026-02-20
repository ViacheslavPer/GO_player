package models

type Song struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

package model

type Column struct {
	Title string `json:"title"`
	Tasks []Task `json:"tasks"`
}

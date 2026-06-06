package model

type MigrationProgress struct {
	Stage              string `json:"stage"`
	Current            string `json:"current"`
	Completed          int    `json:"completed"`
	Total              int    `json:"total"`
	Percent            int    `json:"percent"`
	EstimatedRemaining int    `json:"estimatedRemaining"`
}

package api

type ExperimentalFeatureFlag struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Milestone   string `json:"milestone"`
	Enabled     bool   `json:"enabled"`
}

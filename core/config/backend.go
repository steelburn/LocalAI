package config

// Backend represents a backend configuration
type Backend struct {
	Name        string   `json:"name" yaml:"name"`
	Type        string   `json:"type" yaml:"type"`
	Description string   `json:"description" yaml:"description"`
	URI         string   `json:"uri" yaml:"uri"`
	Icon        string   `json:"icon" yaml:"icon"`
	License     string   `json:"license" yaml:"license"`
	Tags        []string `json:"tags" yaml:"tags"`
}


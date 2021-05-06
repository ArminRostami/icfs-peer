package domain

type UserConfig struct {
	Bootstrap string `json:"bootstrap"`
	SwarmKey  string `json:"swarm_key"`
}

package channel

type ChannelConfig struct {
	Name    string   `json:"name"`
	Owner   string   `json:"owner"`
	Admins  []string `json:"admins"`
	Members []string `json:"members"`
}

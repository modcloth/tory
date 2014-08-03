package tory

type meta struct {
	Hostvars map[string]map[string]interface{} `json:"hostvars"`
}

func newMeta() *meta {
	return &meta{
		Hostvars: map[string]map[string]interface{}{},
	}
}

func (m *meta) AddHostvar(hostname, key string, value interface{}) {
	if _, ok := m.Hostvars[hostname]; !ok {
		m.Hostvars[hostname] = map[string]interface{}{}
	}

	m.Hostvars[hostname][key] = value
}

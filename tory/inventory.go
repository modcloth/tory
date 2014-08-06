package tory

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"
)

var (
	groupNameUnsafe = regexp.MustCompile("[^-A-Za-z0-9]")
)

type inventory struct {
	Meta       *meta `json:"_meta"`
	groups     map[string][]string
	groupMutex *sync.Mutex
}

func newInventory() *inventory {
	return &inventory{
		Meta:       newMeta(),
		groups:     map[string][]string{},
		groupMutex: &sync.Mutex{},
	}
}

func (inv *inventory) GetGroup(group string) []string {
	inv.groupMutex.Lock()
	defer inv.groupMutex.Unlock()

	if g, ok := inv.groups[group]; ok {
		return g
	}

	return nil
}

func (inv *inventory) AddIPToGroup(group, ip string) {
	sanitizedGroup := groupNameUnsafe.ReplaceAllString(strings.ToLower(group), "_")
	sanitizedGroup = strings.Replace(sanitizedGroup, ".", "_", -1)
	inv.AddIPToGroupUnsanitized(sanitizedGroup, ip)
}

func (inv *inventory) AddIPToGroupUnsanitized(group, ip string) {
	inv.groupMutex.Lock()
	defer inv.groupMutex.Unlock()

	if _, ok := inv.groups[group]; !ok {
		inv.groups[group] = []string{}
	}
	inv.groups[group] = append(inv.groups[group], ip)
}

func (inv *inventory) MarshalJSON() ([]byte, error) {
	serialized := map[string]interface{}{}
	serialized["_meta"] = inv.Meta
	for key, value := range inv.groups {
		serialized[key] = value
	}

	return json.Marshal(serialized)
}

func (inv *inventory) UnmarshalJSON(b []byte) error {
	raw := map[string]json.RawMessage{}
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	for key, value := range raw {
		if key == "_meta" {
			m := &meta{}
			err := json.Unmarshal(value, m)
			if err != nil {
				return err
			}
			inv.Meta = m
		} else {
			group := []string{}
			err := json.Unmarshal(value, &group)
			if err == nil {
				for _, ip := range group {
					inv.AddIPToGroupUnsanitized(key, ip)
				}
			}
		}
	}

	return nil
}

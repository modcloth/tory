package tory

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	groupNameUnsafe = regexp.MustCompile("[^-A-Za-z0-9]")
)

type inventory struct {
	Meta   *meta `json:"_meta"`
	groups map[string][]string
}

func newInventory() *inventory {
	return &inventory{
		Meta:   newMeta(),
		groups: map[string][]string{},
	}
}

func (inv *inventory) AddIPToGroup(group, ip string) {
	sanitizedGroup := groupNameUnsafe.ReplaceAllString(strings.ToLower(group), "_")
	sanitizedGroup = strings.Replace(sanitizedGroup, ".", "_", -1)
	inv.AddIPToGroupUnsanitized(sanitizedGroup, ip)
}

func (inv *inventory) AddIPToGroupUnsanitized(group, ip string) {
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

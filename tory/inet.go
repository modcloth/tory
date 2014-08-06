package tory

import (
	"database/sql/driver"
	"strings"
)

type inet struct {
	Addr   string
	Subnet string
}

func (i *inet) Scan(value interface{}) error {
	if value == nil {
		i.Addr = ""
		i.Subnet = ""
		return nil
	}

	strValue := string(value.([]byte))
	parts := strings.SplitN(strValue, "/", 2)
	if len(parts) == 1 {
		i.Addr = parts[0]
	} else if len(parts) > 1 {
		i.Addr = parts[0]
		i.Subnet = parts[1]
	}

	return nil
}

func (i *inet) Value() (driver.Value, error) {
	if i.Addr == "" {
		return nil, nil
	}

	return []byte(i.String()), nil
}

func (i *inet) String() string {
	if i.Addr == "" {
		return ""
	}

	strValue := i.Addr
	if i.Subnet != "" {
		strValue = strValue + "/" + i.Subnet
	}

	return strValue
}

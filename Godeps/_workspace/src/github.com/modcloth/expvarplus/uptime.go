package expvarplus

import (
	"expvar"
	"fmt"
	"time"
)

var (
	startTime = time.Now().UTC()
)

func init() {
	expvar.Publish("uptime", expvar.Func(uptime))
}

func uptime() interface{} {
	now := time.Now().UTC()
	uptimeDur := now.Sub(startTime)

	return map[string]interface{}{
		"start_time":  startTime,
		"uptime":      uptimeDur.String(),
		"uptime_ms":   fmt.Sprintf("%d", uptimeDur.Nanoseconds()/1000000),
		"server_time": now,
	}
}

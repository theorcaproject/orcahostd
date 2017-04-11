package logreceiver

import "orcahostd/client"

type LogSender interface {
	Send(logs map[string]client.Logs)
}

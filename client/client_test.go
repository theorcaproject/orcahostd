package client

import (
)
import (
	"testing"
	"time"
)

func TestPlan__Plan_HostWithFailedAppsAndErrors_Terminated(t *testing.T){
	client := Client{}
	client.Init()

	for {
		client.GetAppState()
		time.Sleep(1)
	}
}
package client

import (
)
import (
	"testing"
	"time"
	"net/http"
)

func TestPlan__Plan_HostWithFailedAppsAndErrors_Terminated(t *testing.T){
	for {
		_, err := http.Get("http://mirror.sphirewall.net")
		if err != nil {
			// handle error
		}
		time.Sleep(2 * time.Second)
	}

}
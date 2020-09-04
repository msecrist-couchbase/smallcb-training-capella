package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

var (
	statsM        sync.Mutex // Protects the stats.
	statsCounters = map[string]uint64{}
	statsInfos    = map[string]string{}
)

func StatsInc(name string) {
	statsM.Lock()
	statsCounters[name] += 1
	statsM.Unlock()
}

func StatsInfo(name, info string) {
	statsM.Lock()
	statsInfos[name] = info
	statsM.Unlock()
}

// ------------------------------------------------

func HttpHandleAdminStats(w http.ResponseWriter, r *http.Request) {
	statsM.Lock()

	stats := map[string]interface{}{
		"counters": statsCounters,
		"infos":    statsInfos,
	}

	result, err := json.MarshalIndent(stats, "", " ")

	statsM.Unlock()

	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", HttpHandleAdminStats, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: HttpHandleAdminStats, err: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(result)
}

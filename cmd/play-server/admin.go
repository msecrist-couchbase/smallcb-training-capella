package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

var (
	statsM     sync.Mutex // Protects the stats.
	statsNums  = map[string]uint64{}
	statsInfos = map[string]string{}
)

func StatsNumInc(name string) {
	statsM.Lock()
	statsNums[name] += 1
	statsM.Unlock()
}

func StatsNum(name string, cb func(uint64) uint64) {
	statsM.Lock()
	statsNums[name] = cb(statsNums[name])
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
		"nums":  statsNums,
		"infos": statsInfos,
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

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

// Atomically increment a statsNums entry by 1.
func StatsNumInc(name string) {
	statsM.Lock()
	statsNums[name] += 1
	statsM.Unlock()
}

// Atomically invoke a read-update callback on a statsNums entry.
func StatsNum(name string, cb func(uint64) uint64) {
	statsM.Lock()
	statsNums[name] = cb(statsNums[name])
	statsM.Unlock()
}

// Atomically set a statsInfos entry to a given string.
func StatsInfo(name, entry string) {
	statsM.Lock()
	statsInfos[name] = entry
	statsM.Unlock()
}

// ------------------------------------------------

func HttpHandleAdminStats(w http.ResponseWriter, r *http.Request) {
	sessionsCount, sessionsCountWithContainer := sessions.Count()

	statsM.Lock()

	statsNums["sessions.count:cur"] = sessionsCount
	statsNums["sessions.countWithContainer:cur"] = sessionsCountWithContainer

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

// ------------------------------------------------

func HttpHandleAdminSessionsReleaseContainers(
	w http.ResponseWriter, r *http.Request) {
	n := sessions.ReleaseContainers(-1)

	w.Header().Set("Content-Type", "application/json")

	j, _ := json.MarshalIndent(map[string]interface{}{
		"status": "ok", "released": n,
	}, "", " ")

	w.Write(j)
}

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	statsM     sync.Mutex // Protects the stats.
	statsNums  = map[string]uint64{}
	statsInfos = map[string]string{}
	statsHists = [][]*StatsHist{[]*StatsHist{}}
)

type StatsHist struct {
	At   string
	Nums map[string]uint64
}

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

	log.Printf("%s: %s", name, entry)
}

// ------------------------------------------------

func StatsRefresh() {
	sessionsCount, sessionsCountWithContainer := sessions.Count()

	statsM.Lock()

	statsNums["sessions.count:cur"] = sessionsCount
	statsNums["sessions.countWithContainer:cur"] = sessionsCountWithContainer

	statsInfos["uptime"] = time.Now().Sub(StartTime).String()

	statsM.Unlock()
}

// ------------------------------------------------

func StatsHistsRun(sampleEvery time.Duration) {
	if sampleEvery > time.Minute {
		panic("sampleEvery too large")
	}

	samplesPerMinute := int(time.Minute / sampleEvery)

	var levelSizes = []int{samplesPerMinute, 60, 24, 10}

	for t := range time.Tick(sampleEvery) {
		StatsRefresh()

		statsHists = StatsHistsSample(statsHists, levelSizes, t)
	}
}

func StatsHistsSample(statsHists [][]*StatsHist,
	levelSizes []int, t time.Time) [][]*StatsHist {
	h := &StatsHist{
		At:   t.Format("2006-01-02T15:04:05"),
		Nums: map[string]uint64{},
	}

	statsM.Lock()

	for k, v := range statsNums {
		h.Nums[k] = v
	}

	statsHists[0] = append(statsHists[0], h)

	statsHists = StatsHistsPromoteLOCKED(statsHists, levelSizes)

	statsM.Unlock()

	return statsHists
}

func StatsHistsPromoteLOCKED(statsHists [][]*StatsHist,
	levelSizes []int) [][]*StatsHist {
	for level := 0; level < len(statsHists); level++ {
		levelSamples := statsHists[level]

		var levelSize int
		if level < len(levelSizes) {
			levelSize = levelSizes[level]
		} else {
			levelSize = levelSizes[len(levelSizes)-1]
		}

		if len(levelSamples) >= levelSize*2 {
			copy(levelSamples[0:levelSize], levelSamples[levelSize:])
			levelSamples = levelSamples[0:levelSize]
			statsHists[level] = levelSamples
		}

		if len(levelSamples) > 0 && len(levelSamples) == levelSize {
			lastSample := levelSamples[len(levelSamples)-1]

			h := &StatsHist{
				At:   lastSample.At,
				Nums: map[string]uint64{},
			}

			for k, v := range lastSample.Nums {
				if !strings.HasSuffix(k, ":cur") &&
					!strings.HasSuffix(k, ":min") &&
					!strings.HasSuffix(k, ":max") {
					h.Nums[k] = v
				}
			}

			for len(statsHists) <= level+1 {
				statsHists = append(statsHists, nil)
			}

			statsHists[level+1] = append(statsHists[level+1], h)
		} else {
			// We didn't add a sample to level+1, so break.
			break
		}
	}

	return statsHists
}

// ------------------------------------------------

func HttpHandleAdminStats(w http.ResponseWriter, r *http.Request) {
	StatsRefresh()

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

// ------------------------------------------------

func HttpHandleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	StatsRefresh()

	keys := map[string]struct{}{}

	statsM.Lock()

	rhists := make([][]*StatsHist, 0, len(statsHists))

	for _, a := range statsHists {
		r := make([]*StatsHist, 0, len(a))
		for j := len(a) - 1; j >= 0; j-- { // Reverse.
			r = append(r, a[j])
		}

		if len(a) > 0 {
			for k := range a[0].Nums {
				keys[k] = struct{}{}
			}
		}

		rhists = append(rhists, r)
	}

	keysArr := make([]string, 0, len(keys))
	for key := range keys {
		keysArr = append(keysArr, key)
	}

	sort.Strings(keysArr)

	statsInfosJ, _ := json.Marshal(statsInfos)

	data := map[string]interface{}{
		"keys":  keysArr,
		"hists": rhists,
		"infos": string(statsInfosJ),
	}

	statsM.Unlock()

	template.Must(template.ParseFiles(
		*staticDir+"/admin-dashboard.html.template")).Execute(w, data)
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

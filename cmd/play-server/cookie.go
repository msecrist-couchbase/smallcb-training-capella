package main

import "sync"

// Protects the cookies map.
var cookiesM sync.Mutex

var cookies = map[string]string{}

func CookiesGet(c string) string {
	cookiesM.Lock()
	v := cookies[c]
	cookiesM.Unlock()
	return v
}

func CookiesSet(c string, v string) {
	cookiesM.Lock()
	cookies[c] = v
	cookiesM.Unlock()
}

func CookiesRemove(cs []string) {
	cookiesM.Lock()
	for _, c := range cs {
		delete(cookies, c)
	}
	cookiesM.Unlock()
}

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

// Get the target IP address of target cluster and add it to NAT route table
func addSrvRoute(host string) {
	updateSrvRoute(host, "add")
}

func delSrvRoute(host string) {
	updateSrvRoute(host, "del")
}

func checkRoute(host string) bool {
	//log.Printf("route -n")
	cmd := exec.Command("route", "-n")
	out, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			log.Printf("Installing net-tools")
			cmd = exec.Command("sudo", "apt-get", "install", "-y", "net-tools")
			_, err := cmd.Output()
			if err != nil {
				log.Printf("err=%v", err)
			}
		} else {
			log.Printf("out=%s, err=%v", out, err)
			return false
		}
	} else {
		if strings.Contains(string(out), host) {
			return true
		} else {
			//fmt.Printf("%s", out)
			return false
		}
	}
	return checkRoute(host)
}

func updateSrvRoute(host string, operation string) {
	sHost := strings.ReplaceAll(host, "couchbase://", "")
	sHost = strings.ReplaceAll(sHost, "couchbases://", "")
	hostName := strings.Split(sHost, ":")[0]
	hostName = strings.Split(hostName, "?")[0]
	cname, srvs, err := net.LookupSRV("couchbases", "tcp", hostName)
	if err != nil {
		log.Printf("err=%v", err)
		return
	}

	fmt.Printf("\ncname: %s \n\n", cname)

	for _, srv := range srvs {
		fmt.Printf("%v:%v:%d:%d\n", srv.Target, srv.Port, srv.Priority, srv.Weight)
		ips, _ := net.LookupIP(srv.Target)
		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				fmt.Println("IPv4: ", ipv4)
				if !checkRoute(ipv4.String()) {
					if operation == "add" {
						log.Printf("route add -host %s gw %s", ipv4.String(), *natPrivateIP)
						cmd := exec.Command("sudo", "route", "add",
							"-host", ipv4.String(), "gw", *natPrivateIP)

						out, err1 := cmd.Output()
						if err1 != nil {
							fmt.Errorf("routeadd, out: %s, err: %v", string(out), err1)
						}
					}
				} else {
					//fmt.Println("Route to " + ipv4.String() + " already exists")
					if operation == "del" {
						log.Printf("route del -host %s gw %s", ipv4.String(), *natPrivateIP)
						cmd := exec.Command("sudo", "route", "del",
							"-host", ipv4.String(), "gw", *natPrivateIP)
						out, err1 := cmd.Output()
						if err1 != nil {
							fmt.Errorf("routedel, out: %s, err: %v", string(out), err1)
						}
					}
				}
			}
		}

	}
}

type cbVersions struct {
	ImplVersion  string       `json:"implementationVersion"`
	CompVersions compVersions `json:"componentsVersion"`
}
type compVersions struct {
	Sasl       string `json:"sasl"`
	NsServer   string `json:"ns_server"`
	Inets      string `json:"inets"`
	OSMonitor  string `json:"os_mon"`
	Chronicle  string `json:"chronicle"`
	Ale        string `json:"ale"`
	Crypto     string `json:"crypto"`
	Stdlib     string `json:"stdlib"`
	PublicKey  string `json:"public_key"`
	SSLVersion string `json:"ssl"`
	Lhttpc     string `json:"lhttpc"`
	Asn1       string `json:"asn1"`
	Kernel     string `json:"kernel"`
}

func GetDBHostFromURL(dburl string) string {
	sHost := strings.ReplaceAll(dburl, "couchbase://", "")
	sHost = strings.ReplaceAll(sHost, "couchbases://", "")
	hostName := strings.Split(sHost, ":")[0]
	hostName = strings.Split(hostName, "?")[0]
	return hostName
}
func CheckDBAccess(dburl string) (string, string, string) {
	hostName := GetDBHostFromURL(dburl)
	Status := "not accessible"
	Version := ""
	IPv4 := ""
	_, srvs, err := net.LookupSRV("couchbases", "tcp", hostName)
	if err != nil {
		Status = "invalid or not reachable"
	} else {
		ip, err := net.LookupIP(srvs[0].Target)
		httpClient := http.Client{
			Timeout: 3 * time.Second,
		}
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		IPv4 = ip[0].To4().String()
		resp, err := httpClient.Get("https://" + IPv4 + ":18091/versions")
		if err != nil {
			Status = "not accessible"
			log.Printf("err=%v", err)
			return Status, Version, IPv4
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			Status = "not accessible"
		} else {
			Status = "OK"
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("err=%v", err)
			}
			var result cbVersions
			if err := json.Unmarshal(body, &result); err != nil {
				fmt.Println("Can not unmarshal cbVersions JSON")
			}
			Version = result.ImplVersion
		}
	}
	log.Printf("Status=%s, Version=%s, IP=%s", Status, Version, IPv4)
	return Status, Version, IPv4
}

func CheckDBUserAccess(dbHost string, dbUser string, dbPwd string) string {
	Status := "not accessible"
	httpClient := http.Client{
		Timeout: 3 * time.Second,
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	resp, err := httpClient.Get("https://" + dbUser + ":" + url.QueryEscape(dbPwd) + "@" + dbHost + ":18091/pools")
	if err != nil {
		Status = "not accessible"
		log.Printf("err=%v", err)
		return Status
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		Status = "not accessible"
	} else {
		Status = "OK"
	}
	return Status
}

func CheckDBUserSampleAccess(dbHost string, dbUser string, dbPwd string, sample string) string {
	Status := "not accessible"
	httpClient := http.Client{
		Timeout: 3 * time.Second,
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	resp, err := httpClient.Get("https://" + dbUser + ":" + url.QueryEscape(dbPwd) + "@" + dbHost + ":18091/pools/default/buckets/" + sample)
	if err != nil {
		Status = "not accessible"
		log.Printf("err=%v", err)
		return Status
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		Status = "not accessible"
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("err=%v", err)
		}
		if strings.Contains(string(body), "resource not found") {
			Status = "not accessible"
		} else {
			Status = "OK"
		}
	}
	return Status
}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	log.Println(localAddr.String())
	return localAddr.IP
}

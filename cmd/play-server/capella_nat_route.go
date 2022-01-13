package main

import (
	"bytes"
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

func GetDBSrvHostFromURL(dburl string) string {
	sHost := strings.ReplaceAll(dburl, "couchbase://", "")
	sHost = strings.ReplaceAll(sHost, "couchbases://", "")
	hostName := strings.Split(sHost, ":")[0]
	hostName = strings.Split(hostName, "?")[0]
	_, srvs, err := net.LookupSRV("couchbases", "tcp", hostName)
	if err != nil {
		log.Printf("err=%v", err)
		return ""
	}
	return strings.TrimSuffix(srvs[0].Target, ".")
}

func GetDBHostIP(dburl string) string {
	hostName := GetDBHostFromURL(dburl)
	IPv4 := ""
	_, srvs, err := net.LookupSRV("couchbases", "tcp", hostName)
	if err != nil {
		return ""
	} else {
		ip, err := net.LookupIP(srvs[0].Target)
		if err != nil {
			log.Printf("err=%v", err)
			return ""
		}
		IPv4 = ip[0].To4().String()
	}
	return IPv4
}

func GetDBHostAllIPs(dburl string) []string {
	hostName := GetDBHostFromURL(dburl)
	IPv4 := []string{}
	_, srvs, err := net.LookupSRV("couchbases", "tcp", hostName)
	if err != nil {
		return IPv4
	} else {
		i := 0
		IPv4 = make([]string, len(srvs))
		for _, srv := range srvs {
			ip, err := net.LookupIP(srv.Target)
			if err != nil {
				log.Printf("err=%v", err)
				return IPv4
			}
			IPv4[i] = ip[0].To4().String()
			i += 1
		}
	}
	return IPv4
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

	resp, err := httpClient.Get("https://" + url.QueryEscape(dbUser) + ":" + url.QueryEscape(dbPwd) + "@" + dbHost + ":18091/pools")
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

	resp, err := httpClient.Get("https://" + url.QueryEscape(dbUser) + ":" + url.QueryEscape(dbPwd) + "@" + dbHost + ":18091/pools/default/buckets/" + sample)
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

func CheckAndCreateFtsIndex(indexName string, dbHost string, dbUser string, dbPwd string) {
	if CheckFtsIndex(indexName, dbHost, dbUser, dbPwd) != "OK" {
		CreateFtsIndex(indexName, dbHost, dbUser, dbPwd)
	}
}

// Create FTS search index
func CreateFtsIndex(indexName string, dbHost string, dbUser string, dbPwd string) string {

	Status := "fts index"
	httpClient := http.Client{
		Timeout: 60 * time.Second,
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	fts_index_json := `{
		"name": "` + indexName + `",
		"type": "fulltext-index",
		"params": {
		 "mapping": {
		  "default_mapping": {
		   "enabled": true,
		   "dynamic": true
		  },
		  "default_type": "_default",
		  "default_analyzer": "standard",
		  "default_datetime_parser": "dateTimeOptional",
		  "default_field": "_all",
		  "store_dynamic": false,
		  "index_dynamic": true,
		  "docvalues_dynamic": false
		 },
		 "store": {
		  "indexType": "scorch",
		  "kvStoreName": ""
		 },
		 "doc_config": {
		  "mode": "type_field",
		  "type_field": "type",
		  "docid_prefix_delim": "",
		  "docid_regexp": ""
		 }
		},
		"sourceType": "couchbase",
		"sourceName": "travel-sample",
		"sourceUUID": "",
		"sourceParams": {},
		"planParams": {
		 "maxPartitionsPerPIndex": 1,
		 "numReplicas": 0,
		 "indexPartitions": 1
		},
		"uuid": ""
	   }`
	//fmt.Println("Running --- PUT https://" +
	//	url.QueryEscape(dbUser) + ":" + url.QueryEscape(dbPwd) + "@" + dbHost + ":18094/api/index/" + indexName)
	req, err := http.NewRequest(http.MethodPut, "https://"+
		url.QueryEscape(dbUser)+":"+url.QueryEscape(dbPwd)+"@"+dbHost+":18094/api/index/"+indexName,
		bytes.NewBuffer([]byte(fts_index_json)))
	if err != nil {
		Status = "fts index creation failed"
		log.Printf("err=%v", err)
		return Status
	}
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("err=%v", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("err=%v", err)
		return "not able to create fts index"
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		Status = "not able to create fts index"
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("err=%v", err)
		}
		if !strings.Contains(string(body), "\"status\":\"ok\"") {
			Status = "not able to create fts index"
		} else {
			Status = "OK"
		}
	}
	// TBD: yet to determine when fts index is going to be finished.
	time.Sleep(10 * time.Second)
	return Status

}

// Create FTS search index
func CheckFtsIndex(indexName string, dbHost string, dbUser string, dbPwd string) string {
	Status := "fts index"
	httpClient := http.Client{
		Timeout: 60 * time.Second,
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	//fmt.Println("Running --- GET https://" +
	//	url.QueryEscape(dbUser) + ":" + url.QueryEscape(dbPwd) + "@" + dbHost + ":18094/api/index/" + indexName)
	resp, err := httpClient.Get("https://" +
		url.QueryEscape(dbUser) + ":" + url.QueryEscape(dbPwd) + "@" + dbHost + ":18094/api/index/" + indexName)
	if err != nil {
		Status = "fts index get failed"
		log.Printf("err=%v", err)
		return Status
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		Status = "not able to get fts index"
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("err=%v", err)
		}
		if !strings.Contains(string(body), "\"status\":\"ok\"") {
			Status = "No fts index found! " + indexName
		} else {
			Status = "OK"
		}
	}
	return Status

}

// Get the AWS instance ID
func GetRunningHostInstanceId() string {
	cmd := exec.Command("curl", "http://169.254.169.254/latest/meta-data/instance-id")
	out, _ := cmd.Output()
	return string(out)
}

// Add the egress security group and rule
func SetEgressToDB(dburl string, email string) string {
	TargetIP := GetDBHostAllIPs(dburl)
	return SetEgress("POST", strings.Join(TargetIP[:], ","), email)
}

// Remove the egress security group and rule
func UnsetEgressToDB(dburl string) string {
	TargetIP := GetDBHostAllIPs(dburl)
	return SetEgress("DELETE", strings.Join(TargetIP[:], ","), "")
}

// Create egress security group
func SetEgress(reqMethod string, ipAddress string, ipAddressOwner string) string {

	Status := "egress security group"
	httpClient := http.Client{
		Timeout: 60 * time.Second,
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	instanceId := GetRunningHostInstanceId()
	payload_json := `{
		"capella_cluster_ip": "` + ipAddress + `",
		"capella_cluster_owner": "` + ipAddressOwner + `",
		"instance_id":  "` + instanceId + `"
	   }`
	//curl -X POST -H "Content-Type: application/json" -d '{"capella_cluster_owner": "jmunta@couchbase.com",
	// "capella_cluster_ip": "35.85.153.221", "instance_id": "i-09571effd432944e0"}' http://betaegresshandlertest-1870274407.us-west-1.elb.amazonaws.com/
	req, err := http.NewRequest(reqMethod, *egressHandlerUrl,
		bytes.NewBuffer([]byte(payload_json)))
	if err != nil {
		Status = "egress " + reqMethod + " failed"
		log.Printf("err=%v", err)
		return Status
	}
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("err=%v", err)
	}
	resp, err := httpClient.Do(req)
	log.Printf("resp=%v", resp)
	if err != nil {
		log.Printf("err=%v", err)
		return "not able to " + reqMethod + " egress security group"
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		Status = "not able to " + reqMethod + " egress security group"
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("err=%v", err)
		}
		if strings.Contains(string(body), "\"status\":\"error\"") {
			Status = "not able to " + reqMethod + " egress security group"
		} else {
			Status = "OK"
		}
	}
	// TBD: yet to determine if this needs some wait.
	time.Sleep(2 * time.Second)
	return Status

}

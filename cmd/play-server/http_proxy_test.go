package main

import "encoding/json"
import "fmt"
import "testing"

func TestRemapJson(t *testing.T) {
	// "pools/default/bs/beer-sample" ==>
	j := `
         {"rev":40,"name":"beer-sample","uri":"...",
          "nodes":[
           {"couchApiBase":"http://$HOST:8092/beer-sample%2B56ac22b352fb084e3bfc27114e6b343a",
            "hostname":"$HOST:8091",
            "ports":{"direct":11210}}],
          "nodesExt":[
           {"services":
            {"mgmt":8091,"mgmtSSL":18091,
             "fts":8094,"ftsSSL":18094,
             "ftsGRPC":9130,"ftsGRPCSSL":19130,
             "indexAdmin":9100,"indexScan":9101,"indexHttp":9102,
             "indexStreamInit":9103,"indexStreamCatchup":9104,"indexStreamMaint":9105,
             "indexHttps":19102,
             "kv":11210,"kvSSL":11207,
             "capi":8092,"capiSSL":18092,
             "projector":9999,
             "n1ql":8093,"n1qlSSL":18093},
           "thisNode":true}],
          "nodeLocator":"vbucket",
          "uuid":"56ac22b352fb084e3bfc27114e6b343a",
          "ddocs":{"uri":"/pools/default/buckets/beer-sample/ddocs"},
          "vBucketServerMap":
           {"hashAlgorithm":"CRC","numReplicas":0,"serverList":["$HOST:11210"],
            "vBucketMap":[[0],[0],[0],[0],[0],[0],[0],[0]]},
          "bucketCapabilitiesVer":"",
          "bucketCapabilities":
           ["durableWrite","tombstonedUserXAttrs","couchapi","dcp",
            "cbhello","touch","cccp","xdcrCheckpointing","nodesExt","xattr"],
          "clusterCapabilitiesVer":[1,0],
          "clusterCapabilities":{"n1ql":["enhancedPreparedStatements"]}}`

	var m map[string]interface{}

	err := json.Unmarshal([]byte(j), &m)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	s := &jsonStreamer{
		remap:     true,
		portMap:   PortMap,
		portStart: 10000,
	}

	s.RemapJson(m)

	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	fmt.Printf("%s\n", b)
}

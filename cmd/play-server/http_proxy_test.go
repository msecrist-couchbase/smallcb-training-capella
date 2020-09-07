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
            "hostname":"172.2.0.3:8091",
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

	s := &JsonRemapper{
		host:      "127.0.0.1",
		portMap:   PortMap,
		portStart: 10000,
	}

	s.RemapJson(m)

	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if string(b) != `{"bucketCapabilities":["durableWrite","tombstonedUserXAttrs","couchapi","dcp","cbhello","touch","cccp","xdcrCheckpointing","nodesExt","xattr"],"bucketCapabilitiesVer":"","clusterCapabilities":{"n1ql":["enhancedPreparedStatements"]},"clusterCapabilitiesVer":[1,0],"ddocs":{"uri":"/pools/default/buckets/beer-sample/ddocs"},"name":"beer-sample","nodeLocator":"vbucket","nodes":[{"couchApiBase":"http://$HOST:8092/beer-sample%2B56ac22b352fb084e3bfc27114e6b343a","hostname":"127.0.0.1:8091","ports":{"direct":11210}}],"nodesExt":[{"services":{"capi":10002,"capiSSL":10012,"fts":10004,"ftsGRPC":9130,"ftsGRPCSSL":19130,"ftsSSL":10014,"indexAdmin":9100,"indexHttp":9102,"indexHttps":19102,"indexScan":9101,"indexStreamCatchup":9104,"indexStreamInit":9103,"indexStreamMaint":9105,"kv":10030,"kvSSL":10027,"mgmt":10001,"mgmtSSL":10011,"n1ql":10003,"n1qlSSL":10013,"projector":9999},"thisNode":true}],"rev":40,"uri":"...","uuid":"56ac22b352fb084e3bfc27114e6b343a","vBucketServerMap":{"hashAlgorithm":"CRC","numReplicas":0,"serverList":["$HOST:10030"],"vBucketMap":[[0],[0],[0],[0],[0],[0],[0],[0]]}}` {
		t.Errorf("%s did the match expected", b)
	}

	fmt.Printf("result: %s\n", b)
}

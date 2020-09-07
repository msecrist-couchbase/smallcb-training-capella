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

	var m interface{}

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

	expected := `{"bucketCapabilities":["durableWrite","tombstonedUserXAttrs","couchapi","dcp","cbhello","touch","cccp","xdcrCheckpointing","nodesExt","xattr"],"bucketCapabilitiesVer":"","clusterCapabilities":{"n1ql":["enhancedPreparedStatements"]},"clusterCapabilitiesVer":[1,0],"ddocs":{"uri":"/pools/default/buckets/beer-sample/ddocs"},"name":"beer-sample","nodeLocator":"vbucket","nodes":[{"couchApiBase":"http://$HOST:8092/beer-sample%2B56ac22b352fb084e3bfc27114e6b343a","hostname":"127.0.0.1:8091","ports":{"direct":11210}}],"nodesExt":[{"services":{"capi":10002,"capiSSL":10012,"fts":10004,"ftsGRPC":9130,"ftsGRPCSSL":19130,"ftsSSL":10014,"indexAdmin":9100,"indexHttp":9102,"indexHttps":19102,"indexScan":9101,"indexStreamCatchup":9104,"indexStreamInit":9103,"indexStreamMaint":9105,"kv":10030,"kvSSL":10027,"mgmt":10001,"mgmtSSL":10011,"n1ql":10003,"n1qlSSL":10013,"projector":9999},"thisNode":true}],"rev":40,"uri":"...","uuid":"56ac22b352fb084e3bfc27114e6b343a","vBucketServerMap":{"hashAlgorithm":"CRC","numReplicas":0,"serverList":["$HOST:10030"],"vBucketMap":[[0],[0],[0],[0],[0],[0],[0],[0]]}}`
	if string(b) != expected {
		t.Errorf("result:\n%s\ndid the match expected\n%s", b, expected)
	}

	fmt.Printf("result: %s\n", b)
}

func TestRemapJsonBuckets(t *testing.T) {
	j := `[{"name":"beer-sample","uuid":"56ac22b352fb084e3bfc27114e6b343a","bucketType":"membase","authType":"sasl","uri":"/pools/default/buckets/beer-sample?bucket_uuid=56ac22b352fb084e3bfc27114e6b343a","streamingUri":"/pools/default/bucketsStreaming/beer-sample?bucket_uuid=56ac22b352fb084e3bfc27114e6b343a","localRandomKeyUri":"/pools/default/buckets/beer-sample/localRandomKey","controllers":{"flush":"/pools/default/buckets/beer-sample/controller/doFlush","compactAll":"/pools/default/buckets/beer-sample/controller/compactBucket","compactDB":"/pools/default/buckets/beer-sample/controller/compactDatabases","purgeDeletes":"/pools/default/buckets/beer-sample/controller/unsafePurgeBucket","startRecovery":"/pools/default/buckets/beer-sample/controller/startRecovery"},"nodes":[{"couchApiBaseHTTPS":"https://172.17.0.2:18092/beer-sample%2B56ac22b352fb084e3bfc27114e6b343a","couchApiBase":"http://172.17.0.2:8092/beer-sample%2B56ac22b352fb084e3bfc27114e6b343a","systemStats":{"cpu_utilization_rate":3.839441535776615,"cpu_stolen_rate":0,"swap_total":1073737728,"swap_used":187432960,"mem_total":2087469056,"mem_free":918052864,"mem_limit":2087469056,"cpu_cores_available":6,"allocstall":0},"interestingStats":{"cmd_get":0,"couch_docs_actual_disk_size":3604427,"couch_docs_data_size":3357066,"couch_spatial_data_size":0,"couch_spatial_disk_size":0,"couch_views_actual_disk_size":1417514,"couch_views_data_size":1417514,"curr_items":7303,"curr_items_tot":7303,"ep_bg_fetched":0,"get_hits":0,"mem_used":5711864,"ops":0,"vb_active_num_non_resident":0,"vb_replica_curr_items":0},"uptime":"161","memoryTotal":2087469056,"memoryFree":918052864,"mcdMemoryReserved":1592,"mcdMemoryAllocated":1592,"replication":1,"clusterMembership":"active","recoveryType":"none","status":"healthy","otpNode":"ns_1@cb.local","thisNode":true,"hostname":"172.17.0.2:8091","nodeUUID":"fababa342398536b2474a05dda9fc761","clusterCompatibility":393222,"version":"6.6.0-7909-enterprise","os":"x86_64-unknown-linux-gnu","cpuCount":6,"ports":{"direct":11210,"httpsCAPI":18092,"httpsMgmt":18091,"distTCP":21100,"distTLS":21150},"services":["fts","index","kv","n1ql"],"nodeEncryption":false,"configuredHostname":"127.0.0.1:8091","addressFamily":"inet","externalListeners":[{"afamily":"inet","nodeEncryption":false},{"afamily":"inet6","nodeEncryption":false}]}],"stats":{"uri":"/pools/default/buckets/beer-sample/stats","directoryURI":"/pools/default/buckets/beer-sample/statsDirectory","nodeStatsListURI":"/pools/default/buckets/beer-sample/nodes"},"nodeLocator":"vbucket","saslPassword":"e224bf385baa71b6572b9be57513dc47","ddocs":{"uri":"/pools/default/buckets/beer-sample/ddocs"},"replicaIndex":false,"autoCompactionSettings":false,"vBucketServerMap":{"hashAlgorithm":"CRC","numReplicas":0,"serverList":["172.17.0.2:11210"],"vBucketMap":[[0],[0],[0],[0],[0],[0],[0],[0]]},"maxTTL":0,"compressionMode":"passive","replicaNumber":0,"threadsNumber":3,"quota":{"ram":134217728,"rawRAM":134217728},"basicStats":{"quotaPercentUsed":4.255670309066772,"opsPerSec":0,"diskFetches":0,"itemCount":7303,"diskUsed":5021941,"dataUsed":4774580,"memUsed":5711864,"vbActiveNumNonResident":0},"evictionPolicy":"fullEviction","durabilityMinLevel":"none","conflictResolutionType":"seqno","bucketCapabilitiesVer":"","bucketCapabilities":["durableWrite","tombstonedUserXAttrs","couchapi","dcp","cbhello","touch","cccp","xdcrCheckpointing","nodesExt","xattr"]}]`

	var m interface{}

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

	expected := `[{"authType":"sasl","autoCompactionSettings":false,"basicStats":{"dataUsed":4774580,"diskFetches":0,"diskUsed":5021941,"itemCount":7303,"memUsed":5711864,"opsPerSec":0,"quotaPercentUsed":4.255670309066772,"vbActiveNumNonResident":0},"bucketCapabilities":["durableWrite","tombstonedUserXAttrs","couchapi","dcp","cbhello","touch","cccp","xdcrCheckpointing","nodesExt","xattr"],"bucketCapabilitiesVer":"","bucketType":"membase","compressionMode":"passive","conflictResolutionType":"seqno","controllers":{"compactAll":"/pools/default/buckets/beer-sample/controller/compactBucket","compactDB":"/pools/default/buckets/beer-sample/controller/compactDatabases","flush":"/pools/default/buckets/beer-sample/controller/doFlush","purgeDeletes":"/pools/default/buckets/beer-sample/controller/unsafePurgeBucket","startRecovery":"/pools/default/buckets/beer-sample/controller/startRecovery"},"ddocs":{"uri":"/pools/default/buckets/beer-sample/ddocs"},"durabilityMinLevel":"none","evictionPolicy":"fullEviction","localRandomKeyUri":"/pools/default/buckets/beer-sample/localRandomKey","maxTTL":0,"name":"beer-sample","nodeLocator":"vbucket","nodes":[{"addressFamily":"inet","clusterCompatibility":393222,"clusterMembership":"active","configuredHostname":"127.0.0.1:8091","couchApiBase":"http://172.17.0.2:8092/beer-sample%2B56ac22b352fb084e3bfc27114e6b343a","couchApiBaseHTTPS":"https://172.17.0.2:18092/beer-sample%2B56ac22b352fb084e3bfc27114e6b343a","cpuCount":6,"externalListeners":[{"afamily":"inet","nodeEncryption":false},{"afamily":"inet6","nodeEncryption":false}],"hostname":"127.0.0.1:8091","interestingStats":{"cmd_get":0,"couch_docs_actual_disk_size":3604427,"couch_docs_data_size":3357066,"couch_spatial_data_size":0,"couch_spatial_disk_size":0,"couch_views_actual_disk_size":1417514,"couch_views_data_size":1417514,"curr_items":7303,"curr_items_tot":7303,"ep_bg_fetched":0,"get_hits":0,"mem_used":5711864,"ops":0,"vb_active_num_non_resident":0,"vb_replica_curr_items":0},"mcdMemoryAllocated":1592,"mcdMemoryReserved":1592,"memoryFree":918052864,"memoryTotal":2087469056,"nodeEncryption":false,"nodeUUID":"fababa342398536b2474a05dda9fc761","os":"x86_64-unknown-linux-gnu","otpNode":"ns_1@cb.local","ports":{"direct":11210,"distTCP":21100,"distTLS":21150,"httpsCAPI":18092,"httpsMgmt":18091},"recoveryType":"none","replication":1,"services":["fts","index","kv","n1ql"],"status":"healthy","systemStats":{"allocstall":0,"cpu_cores_available":6,"cpu_stolen_rate":0,"cpu_utilization_rate":3.839441535776615,"mem_free":918052864,"mem_limit":2087469056,"mem_total":2087469056,"swap_total":1073737728,"swap_used":187432960},"thisNode":true,"uptime":"161","version":"6.6.0-7909-enterprise"}],"quota":{"ram":134217728,"rawRAM":134217728},"replicaIndex":false,"replicaNumber":0,"saslPassword":"e224bf385baa71b6572b9be57513dc47","stats":{"directoryURI":"/pools/default/buckets/beer-sample/statsDirectory","nodeStatsListURI":"/pools/default/buckets/beer-sample/nodes","uri":"/pools/default/buckets/beer-sample/stats"},"streamingUri":"/pools/default/bucketsStreaming/beer-sample?bucket_uuid=56ac22b352fb084e3bfc27114e6b343a","threadsNumber":3,"uri":"/pools/default/buckets/beer-sample?bucket_uuid=56ac22b352fb084e3bfc27114e6b343a","uuid":"56ac22b352fb084e3bfc27114e6b343a","vBucketServerMap":{"hashAlgorithm":"CRC","numReplicas":0,"serverList":["127.0.0.1:10030"],"vBucketMap":[[0],[0],[0],[0],[0],[0],[0],[0]]}}]`
	if string(b) != expected {
		t.Errorf("result:\n%s\ndid the match expected\n%s", b, expected)
	}

	fmt.Printf("result: %s\n", b)
}

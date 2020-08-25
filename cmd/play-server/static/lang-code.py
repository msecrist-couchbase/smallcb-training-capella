#!/usr/bin/python3

from couchbase.cluster import Cluster, ClusterOptions, QueryOptions
from couchbase_core.cluster import PasswordAuthenticator

pa = PasswordAuthenticator('Administrator', 'password')

cluster = Cluster('couchbase://localhost', ClusterOptions(pa))

bucket = cluster.bucket('beer-sample')

coll = bucket.default_collection()

rv = coll.get('21st_amendment_brewery_cafe')

print(rv.content)


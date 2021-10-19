#!/usr/bin/env cbsh --script
# This is a simple Couchbase cloud helper script
# cbsh --script cloud_helper_cbshell.nu

def log [
    message:any
] {
    let now = (date now | date format '%Y%m%d_%H%M%S.%f')
    let mess = $"($now)|INFO|($message)(char nl)"
    echo $mess | autoview
}

def progress-bar [] {
    let blocks = ["▏" "▎" "▍" "▌" "▋" "▊" "▉" "█"]
    let pb_size = 50
    ansi cursor_off
    echo 1..<$pb_size | each { |cur_size|
        echo 0..7 | each { |tick|
            let idx = ($tick mod 8)
            let cur_block = (echo $blocks | nth $idx)
            $"($cur_block | str lpad -c $blocks.7 -l $cur_size)" | autoview
            $"(ansi -e '1000D')" | autoview
            sleep 500ms
        }
    }
    char newline
    '.'
    ansi cursor_on
}

def progress-bar-wait-for [ cluster-name expected-status ] {
    let blocks = ["▏" "▎" "▍" "▌" "▋" "▊" "▉" "█"]
    let pb_size = 100
    ansi cursor_off
    let cur_size = 10

    let actual-status = $"(clouds clusters-get $cluster-name | get status)"
    if $actual-status != $expected-status {
       echo 1..<$pb_size | each { |cur_size|
        echo 0..7 | each { |tick|
            let idx = ($tick mod 8)
            let cur_block = (echo $blocks | nth $idx)
            $"($cur_block | str lpad -c $blocks.7 -l $cur_size)" | autoview
            $"(ansi -e '1000D')" | autoview
            sleep 500ms
            let actual-status = $"(clouds clusters-get $cluster-name | get status)"
        }
       }
        progress-bar-wait-for $cluster-name $expected-status
    } {
        char newline
        'Done'
        ansi cursor_on
        clouds clusters-get $cluster-name
    }
    
}

def progress-bar-wait-for-not-in [ cluster-name expected-status ] {
    let blocks = ["▏" "▎" "▍" "▌" "▋" "▊" "▉" "█"]
    let pb_size = 50
    ansi cursor_off
    let cur_size = 10

    let actual-status = $"(do -i {clouds clusters-get $cluster-name | get status})"
    if $actual-status == $expected-status {
     echo 1..<$pb_size | each { |cur_size|
        echo 0..7 | each { |tick|
            let idx = ($tick mod 8)
            let cur_block = (echo $blocks | nth $idx)
            $"($cur_block | str lpad -c $blocks.7 -l $cur_size)" | autoview
            $"(ansi -e '1000D')" | autoview
            sleep 500ms
            let actual-status = $"(do -i {clouds clusters-get $cluster-name | get status})"
        }
     }
        progress-bar-wait-for-not-in $cluster-name $expected-status
    } {
        char newline
        'Done'
    }
    ansi cursor_on
}

def cloud-status [] {
    log "clouds"
    clouds

    log "projects"
    projects
    
    log "clusters"
    clouds clusters
    
    clouds clusters | each { clouds clusters-get $it.name }
    
    log "users"
    users
}


def cluster-status [] {
    if ($nu.env | default CLUSTER "" | get CLUSTER | empty?) {
        if ($nu.env | default CMD "" | get CMD | empty?) {
            #log "clusters"
            #clouds clusters
            #clouds clusters | each { clouds clusters-get $it.name }
            log "Usage: 
                CLUSTER_CREATE=<cluster-name> cloud_helper_cbshell.nu  : Creates a new cluster cluster-name
                CLUSTER_DELETE=<cluster-name> cloud_helper_cbshell.nu  : Deletes cluster cluster-name
                CLUSTER=<cluster-name> cloud_helper_cbshell.nu  : Gets details on cluster cluster-name
                CMD=<cmd> cloud_helper_cbshell.nu  : Runs a given cmd
            "
        } {
            let cmd = $nu.env.CMD
            log $"running ($cmd)"
            if $cmd =~ "nu " {
                let c2 = ($cmd | split row ' ' | range 1.. | each { ' ' + $it } | str collect | str trim)
                echo $"cs=($c2)"
                exec ($c2)
            } {
                cbsh -c ($cmd)
            }
        }
    } {
        let cluster-name = $nu.env.CLUSTER
        log $"status for the cluster ($cluster-name)"
        clouds clusters-get $cluster-name
        let cluster-status = $"(do -i {clouds clusters-get $cluster-name | get status})"
        if ($cluster-status == "deploying") {
            progress-bar-wait-for $"($cluster-name)" "ready"
        } {}
    }
}

def cluster-create [] {
    if ($nu.env | default CLUSTER_CREATE "" | get CLUSTER_CREATE | empty?) { 
    } {
        let cluster-name = $nu.env.CLUSTER_CREATE
        let cluster-status = $"(do -i {clouds clusters-get $cluster-name | get status})"
        if ($cluster-status | empty?) {
            log $"Please wait while creating new cluster ($cluster-name) ... ~15 mins"
            let payloadx = $'{"cloudId": "", "projectId": "", "name": "($cluster-name)", "version": "latest", "servers": [{"services": ["data", "query", "index", "search"], "size": 3, "aws": {"ebsSizeGib": 1227, "instanceSize": "r5.xlarge"}}], "supportPackage": {"timezone": "PT", "type": "developerPro"}}'
            clouds clusters-create $'{"cloudId": "", "projectId": "", "name": "($cluster-name)", "version": "latest", "servers": [{"services": ["data", "query", "index", "search"], "size": 3, "aws": {"ebsSizeGib": 1227, "instanceSize": "r5.xlarge"}}], "supportPackage": {"timezone": "PT", "type": "developerPro"}}'
            sleep 1000ms # wait to get started
            clouds clusters-get $cluster-name
            progress-bar-wait-for $"($cluster-name)" "ready"
            let-env CLUSTER_CONFIG = $cluster-name
            cluster-config
        } {
            log $"Cluster ($cluster-name) ($cluster-status)... choose a different name."
        }
    }
    
}

def cluster-delete [] {
    if ($nu.env | default CLUSTER_DELETE "" | get CLUSTER_DELETE | empty?) { 
    } {
        let cluster-name = $nu.env.CLUSTER_DELETE
        let status = $"(do -i {clouds clusters-get $cluster-name | get status})"
        if $"($status)" == "ready" {
            log $"Please wait while deleting existing cluster ($cluster-name) ... ~5 mins"
            clouds clusters-drop $cluster-name
            sleep 2000ms
            clouds clusters-get $cluster-name 
            progress-bar-wait-for-not-in $"($cluster-name)" "destroying"
        } {
            if ($status | empty?) {
                log $"cluster ($cluster-name) doesn't exist."
            } {
                progress-bar-wait-for-not-in $"($cluster-name)" "destroying"
            }
        }
    }
    
}

def cluster-config [
     ] {

    if ($nu.env | default CLUSTER_CONFIG "" | get CLUSTER_CONFIG | empty?) { 
    } {
        let cluster-name = $nu.env.CLUSTER_CONFIG
        log $"configuring the new cluster ($cluster-name)"
        let endpoint-srv = $"(clouds clusters-get $cluster-name | get endpoint_srv)"
        let username = (random chars -l 10)
        let password = $"(random chars -l 17)(random integer 1..100)!"
        let organization = $"(use | get cloud_organization)"
        clusters register $cluster-name $"($endpoint-srv)" $username $"($password)" --cloud-organization $"($organization)" --save
        use cluster $cluster-name
        clusters
        log $"adding database user ($username)"
        users upsert ($username) "data_writer" --password $password --clusters $cluster-name
        users
        log $"adding allowlist 0.0.0.0/0"
        addresses add "0.0.0.0/0" --clusters $cluster-name
        sleep 1000ms
        addresses
        log $"import travel-sample data"
        log $"curl -k -u'<clouduser:clouduser-pwd>' -d '[travel_sample]' https://cb-0000.($endpoint-srv).dp.cloud.couchbase.com:18091/sampleBuckets/install"
        log $"buckets"
        buckets
        log $"Couchbase cluster URL=couchbases://($endpoint-srv), username=($username), password=($password)" 
    }   
}

#cloud-status
cluster-status
cluster-create
cluster-config
cluster-delete

char newline
log "Done"
char newline
ansi cursor_on
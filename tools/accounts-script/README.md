# Accounts Script

A tool that runs a given script against all accounts using an execution checkpoint. 

## Requirements

You need access to an instance capable of loading an execution checkpoint into memory (~500-600GB) for example an instance of type `n2-highmem-96` works well. You need access to said execution checkpoint on that instance as well (for example as a disk).

This tool was tested on vm `mainnet26-execution-leo`.

## Usage example

Assuming this is running on a vm `mainnet26-execution-xyz`, we are given an entire execution snapshot (as opposed to just a checkpoint), we are starting from scratch and all required permissions are given.

```sh
# start vm instance, change zone and project flags as required
gcloud compute instances start mainnet26-execution-xyz --zone=us-central1-f --project=flow-multi-region

# connect via ssh, change zone and project flags as required
gcloud compute ssh --ssh-flag="-A" --tunnel-through-iap --zone=us-central1-f mainnet26-execution-xyz --project=flow-multi-region

sudo su

cd ../..

# check where the snapshot is available (look for ~24TB)
lsblk

mkdir /mnt/snapshot

# this depends on lsblk output
mount /dev/sdc/ /mnt/snapshot 

cd home/your_email@flowfoundation.org

git clone https://github.com/onflow/cadence.git

cd cadence/tools/accounts-script

# adjust flags for use case and setup
go run main.go --checkpoint-dir ../../../../../mnt/snapshot/bootstrap/execution-state/ --chain flow-mainnet --script ./iterate-storage.cdc
```

Example output

```console
root@mainnet26-execution-leo:/home/raymond_zhang_flowfoundation_org/cadence/tools/accounts-script# go run main.go --checkpoint-dir ../../../../../mnt/snapshot/bootstrap/execution-state/ --chain flow-mainnet --script ./iterate_storage.cdc 
17:22:00 INF loading checkpoint(s) from ../../../../../mnt/snapshot/bootstrap/execution-state/
17:26:36 INF the checkpoint is loaded
17:26:36 INF Processing 2 tries
17:26:36 INF created 0 registers from payloads (0 accounts)
17:42:44 INF created 612185649 registers from payloads (35016463 accounts)
17:42:44 INF processing accounts progress 1/35016463 (0.0%) elapsed: 0s, eta 3m55s
17:43:44 INF processing accounts progress 3372000/35016463 (9.6%) elapsed: 1m0s, eta 9m23s
17:43:47 INF processing accounts progress 3501647/35016463 (10.0%) elapsed: 1m3s, eta 9m28s
17:44:47 INF processing accounts progress 6638637/35016463 (19.0%) elapsed: 2m3s, eta 8m46s
17:44:56 INF processing accounts progress 7003293/35016463 (20.0%) elapsed: 2m13s, eta 8m50s
17:45:56 INF processing accounts progress 9951000/35016463 (28.4%) elapsed: 3m13s, eta 8m5s
17:46:07 INF processing accounts progress 10504939/35016463 (30.0%) elapsed: 3m23s, eta 7m54s
17:47:06 INF processing accounts progress 14006585/35016463 (40.0%) elapsed: 4m23s, eta 6m34s
17:48:06 INF processing accounts progress 16723000/35016463 (47.8%) elapsed: 5m23s, eta 5m53s
17:48:21 INF processing accounts progress 17508231/35016463 (50.0%) elapsed: 5m37s, eta 5m37s
17:49:21 INF processing accounts progress 20924000/35016463 (59.8%) elapsed: 6m37s, eta 4m28s
17:49:22 INF processing accounts progress 21009877/35016463 (60.0%) elapsed: 6m39s, eta 4m26s
17:50:22 INF processing accounts progress 24177000/35016463 (69.0%) elapsed: 7m39s, eta 3m26s
17:50:30 INF processing accounts progress 24511523/35016463 (70.0%) elapsed: 7m46s, eta 3m20s
17:51:30 INF processing accounts progress 27816000/35016463 (79.4%) elapsed: 8m46s, eta 2m16s
17:51:33 INF processing accounts progress 28013169/35016463 (80.0%) elapsed: 8m49s, eta 2m12s
17:52:33 INF processing accounts progress 31459000/35016463 (89.8%) elapsed: 9m49s, eta 1m7s
17:52:34 INF processing accounts progress 31514815/35016463 (90.0%) elapsed: 9m50s, eta 1m6s
17:53:34 INF processing accounts progress 34374000/35016463 (98.2%) elapsed: 10m50s, eta 12s
17:53:45 INF processing accounts progress 35016461/35016463 (100.0%) elapsed: 11m2s, eta 0s
17:53:45 INF processing accounts progress 35016463/35016463 (100.0%) total time 11m2s
17:53:45 INF Success
```

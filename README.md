# kube-dangler: K8s Orphaned Pod Finder
Minimal CLI to list potentially dangling (attached to no Service) Pods in Kubernetes

Designed for Site Reliability Engineers (SREs) to identify "dangling" Pods—containers that are running and consuming resources but are not targeted by any Kubernetes Service.

This tool helps you identify:
* **Stranded Deployments:** Services deleted without removing the underlying Deployment.
* **Label Mismatches:** Pods that aren't receiving traffic because their labels don't match the Service selector.
* **Leaked Pods:** Standalone pods created manually or by older operator versions that were never cleaned up.


## Features

* **Context-Aware:** Automatically detects if it's running inside a cluster (using In-Cluster config) or locally (using `~/.kube/config`).
* **Noise Reduction:** Automatically ignores Pods owned by **Jobs** (which are naturally service-less).
* **Namespace Scoping:** Filter by a specific namespace or scan the entire cluster.
* **Safety First:** Kube-* namespaces (e.g. `kube-system`, `kube-public` etc.) excluded from search by default.

## Installation

Build from source:

```bash
git clone https://github.com/AleksanderWWW/kube-dangler.git

cd kube-dangler

go build -o kubedangler main.go

./kubedangler --help
```

From Github Releases:

```shell
export TAG="0.1.0"

wget https://github.com/AleksanderWWW/kube-dangler/releases/download/${TAG}/kubedangler-${TAG}.tar.gz

tar -xzf kubedangler-${TAG}.tar.gz

chmod +x kubedangler

./kubedangler --help
```

Expected output:

```terminaloutput
$ ./kubedangler --help
NAME:
   kubedangler - find potentially dangling Pods (attached to no Service)

USAGE:
   kubedangler [global options]

GLOBAL OPTIONS:
   --namespace string, -n string  namespace to check for dangling pods (default: look through all namespaces)
   --min-age duration             minimal age of potentially dangling pods (default: 1h0m0s)
   --include-kube-ns              whether to also include checking the kube namespaces
   --version                      print version number and exit
   --help, -h                     show help
```

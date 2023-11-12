# Distributed Scheduler for Kubernetes



## Directory structre

```
.
├── bin # binary for scheudler and controller
├── build
│   └── package # Dockerfiles
├── cmd # main files for scheduler and controller
│   ├── controller
│   ├── rand_sched
│   └── scheduler
├── deploy # k8s deployment files
│   ├── bbox
│   │   └── templates
│   ├── elasticsearch
│   ├── fluentd
│   ├── kibana
│   ├── templates
│   └── workload
│       ├── fft
│       └── fibonacci
├── internal # main hyperion implementation
│   ├── configs # scheduler configs
│   │   └── data
│   ├── controller
│   ├── message # Protobuf
│   ├── metrics # prometheus
│   ├── rand_sched
│   ├── scheduler
│   ├── server
│   └── util
├── log
├── measure
│   ├── data
│   ├── graphs
│   ├── old_data
│   ├── profiles
│   └── trace
│       └── 2022-08-23T17-11-56
│           └── trace
└── scripts # scripts to invoke and deploy schedulers and jobs
    └── __pycache__
```
#!/usr/bin/env python3

from kubernetes import client, config
import json

# Configs can be set in Configuration class directly or using helper utility
config.load_kube_config()


stats_name = ['got jobs',
              'initial w',
              'final w' ]

'''
We analyse the following per pod:
  msg rcv per iter
  msg sent per iter
  msg rcv total
  msg sent total
  total time per iter
  compute time per iter
  xchg time per iter
  total time


time unit is in microsecond 1e-6

'''

metrics = {}


v1 = client.CoreV1Api()
ret = v1.list_namespaced_pod(namespace='dist-sched', watch=False)
for i in ret.items:
    if i.metadata.name.startswith("my-scheduler-") or i.metadata.name.startswith("my-controller"):
        # print(f"{i.metadata.name} {i.status.pod_ip}")
        lines = v1.read_namespaced_pod_log(
            name=i.metadata.name, namespace=i.metadata.namespace)

        sched_name = ""
        if i.metadata.name.startswith('my-scheduler-'):
            sched_name = i.metadata.name
        for line in lines.split('\n'):
            if sched_name.startswith('my-sched') and line.startswith('{') and line.find("placement") != -1 \
                and line.find("trial") != -1:
                d = json.loads(line)

                trial = int(d['trial']) - 1
                if len(metrics) < trial + 1:
                    metrics[trial] = {}
                m = metrics[trial]
                if sched_name not in metrics[trial]:
                    m[sched_name] = {}

                for sn in stats_name:
                    if sn in d:
                        m[sched_name][sn] = d[sn]


print(json.dumps(metrics))

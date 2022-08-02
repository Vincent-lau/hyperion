#!/usr/bin/env python3

from kubernetes import client, config
import json

# Configs can be set in Configuration class directly or using helper utility
config.load_kube_config()


stats_name = ['msg rcv total',
              'msg sent total',
              'comp time per iter',
              'xchg time per iter',
              'avg time per iter',
              'total time',
              'total iter']

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
ret = v1.list_pod_for_all_namespaces(watch=False)
for i in ret.items:
    if i.metadata.name.startswith("my-scheduler-") or i.metadata.name.startswith("my-controller"):
        # print(f"{i.metadata.name} {i.status.pod_ip}")
        lines = v1.read_namespaced_pod_log(
            name=i.metadata.name, namespace=i.metadata.namespace)

        name = i.metadata.name
        if name.startswith('my-scheduler-'):
            metrics[name] = {}
        for line in lines.split('\n'):
            if name.startswith('my-sched') and line.startswith('{'):
                d = json.loads(line)
                for sn in stats_name:
                    if sn.endswith('per iter'):
                        if not sn in metrics[name]:
                            metrics[name][sn] = []
                        if sn in d:
                            metrics[name][sn].append(d[sn])
                    else:
                        if sn in d:
                            metrics[name][sn] = d[sn]


print(json.dumps(metrics))

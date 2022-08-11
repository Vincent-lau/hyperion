#!/usr/bin/env python3

import json, datetime, os

from kubernetes import client, config

# Configs can be set in Configuration class directly or using helper utility
config.load_kube_config()


stats_name = ['msg rcv total',
              'msg sent total',
              'comp time per iter',
              'xchg time per iter',
              'avg time per iter',
              'total time',
              'total iter',
              'setup time']

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
            if sched_name.startswith('my-sched') and line.startswith('{') and line.find("metrics") != -1 \
                    and line.find("trial") != -1:
                d = json.loads(line)

                trial = int(d['trial']) - 1
                if len(metrics) < trial + 1:
                    metrics[trial] = {}
                m = metrics[trial]
                if sched_name not in metrics[trial]:
                    m[sched_name] = {}

                for sn in stats_name:
                    # if sn.endswith('per iter'):
                    #     if not sn in m[sched_name]:
                    #         m[sched_name][sn] = []
                    #     if sn in d:
                    #         m[sched_name][sn].append(d[sn])
                    # else:
                    if sn in d:
                        m[sched_name][sn] = d[sn]



name = datetime.datetime.now().isoformat()[:-7].replace(':', '-')
dir = f"measure/data/{name}/"
os.mkdir(dir)

with open(f"{dir}/metrics.json", 'w') as f:
    json.dump(metrics, f)

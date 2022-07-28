#!/usr/bin/env python3

from kubernetes import client, config
import os

# Configs can be set in Configuration class directly or using helper utility
config.load_kube_config()

v1 = client.CoreV1Api()
print("copying profiles from pods")
ret = v1.list_pod_for_all_namespaces(watch=False)

k = 0
for i in ret.items:
    if i.metadata.name.startswith("my-scheduler-"):
      os.system(f'kubectl cp kube-system/{i.metadata.name}:sched.prof profiles/sched{k}.prof')
      print(f'copied from {i.metadata.name} to sched{k}.prof')
      k += 1


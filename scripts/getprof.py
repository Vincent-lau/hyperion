#!/usr/bin/env python3

from kubernetes import client, config
import os, datetime

# Configs can be set in Configuration class directly or using helper utility
config.load_kube_config()

v1 = client.CoreV1Api()
print("copying profiles from pods")
ret = v1.list_pod_for_all_namespaces(watch=False)

k = 0
name = datetime.datetime.now().isoformat()[:-7].replace(':', '-')
dir = f"measure/profiles/{name}/"
os.mkdir(dir)
for i in ret.items:
    if i.metadata.name.startswith("my-scheduler-"):
      os.system(f'kubectl cp {i.metadata.name}:sched.prof {dir}sched{k}.prof')
      print(f'copied from {i.metadata.name} to {dir}sched{k}.prof')
      k += 1
    if k > 5:
      break


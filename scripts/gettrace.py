#!/usr/bin/env python3

from kubernetes import client, config
import os, datetime

# Configs can be set in Configuration class directly or using helper utility
config.load_kube_config()

v1 = client.CoreV1Api()
print("copying traces from pods")
ret = v1.list_pod_for_all_namespaces(watch=False)

k = 0
name = datetime.datetime.now().isoformat()[:-7].replace(':', '-')
dir = f"/local/scratch/sl955/dist_sched/measure/trace/{name}"
os.mkdir(dir)
for i in ret.items:
    if i.metadata.name.startswith("my-scheduler-"):
      os.system(f'kubectl cp {i.metadata.name}:trace/ {dir}/trace/{i.metadata.name}/')
      print(f'copied from {i.metadata.name}:trace/ to {dir}/trace/{i.metadata.name}/')
      k += 1


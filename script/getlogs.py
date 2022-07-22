#!/usr/bin/env python3

from kubernetes import client, config
import os

# Configs can be set in Configuration class directly or using helper utility
config.load_kube_config()

v1 = client.CoreV1Api()
print("Listing pods with their IPs:")
ret = v1.list_pod_for_all_namespaces(watch=False)
for i in ret.items:
    if i.metadata.name.startswith("my-scheduler-") or i.metadata.name.startswith("my-controller"):
      print(f"{i.metadata.name} {i.status.pod_ip}")
      print(v1.read_namespaced_pod_log(name = i.metadata.name, namespace = i.metadata.namespace))
      print('-----------------------------------------------------')

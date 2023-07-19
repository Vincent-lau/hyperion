#!/usr/bin/env python3

import datetime
import json
import os
import argparse

from kubernetes import client, config

# Configs can be set in Configuration class directly or using helper utility
config.load_kube_config()


consensus_stats_name = ['msg rcv total',
                        'msg sent total',
                        'comp time per iter',
                        'xchg time per iter',
                        'avg time per iter',
                        'total time',
                        'total iter',
                        'setup time']

placement_stats_name = [
    "got jobs",
    "initial w",
    "final w",
    "left elements",
    "scheduled elements",
    "time taken",
    "generated jobs"
]

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
name = datetime.datetime.now().isoformat()[:-7].replace(':', '-')
dir = f"measure/data/{name}"
os.mkdir(dir)


def read_pl(pods: int, jobs: int):
    ret = v1.list_namespaced_pod(namespace='dist-sched', watch=False)
    for i in ret.items:
        if i.metadata.name.startswith("my-scheduler-") or i.metadata.name.startswith("my-controller"):
            # print(f"{i.metadata.name} {i.status.pod_ip}")
            lines = v1.read_namespaced_pod_log(
                name=i.metadata.name, namespace=i.metadata.namespace)

            sched_name = ""
            if i.metadata.name.startswith('my-scheduler-') or i.metadata.name.startswith('my-controller'):
                sched_name = i.metadata.name
            for line in lines.split('\n'):
                if line.startswith('{') and line.find("placement") != -1 and line.find("trial") != -1:
                    d = json.loads(line)

                    trial = int(d['trial']) - 1
                    if len(metrics) < trial + 1:
                        metrics[trial] = {}
                    m = metrics[trial]
                    if sched_name not in metrics[trial]:
                        m[sched_name] = {}

                    for sn in placement_stats_name:
                        if sn in d:
                            m[sched_name][sn] = d[sn]
    fname = f"{dir}/placement-{pods}pods-{jobs}jobs.json"

    with open(fname, 'w') as f:
        json.dump(metrics, f)

    print(f"{fname} is created for placement")


def read_con(pods: int, jobs: int):
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

                    for sn in consensus_stats_name:
                        # if sn.endswith('per iter'):
                        #     if not sn in m[sched_name]:
                        #         m[sched_name][sn] = []
                        #     if sn in d:
                        #         m[sched_name][sn].append(d[sn])
                        # else:
                        if sn in d:
                            m[sched_name][sn] = d[sn]

    fname = f"{dir}/consensus-{pods}pods-{jobs}jobs.json"
    with open(fname, 'w') as f:
        json.dump(metrics, f)

    print(f"{fname} is created for consensus")


def read_all_log(pods: int, jobs: int):
    ret = v1.list_namespaced_pod(namespace='dist-sched', watch=False)
    data = ""
    for i in ret.items:
        if i.metadata.name.startswith("my-scheduler-") or i.metadata.name.startswith("my-controller"):
            # print(f"{i.metadata.name} {i.status.pod_ip}")
            lines = v1.read_namespaced_pod_log(
                name=i.metadata.name, namespace=i.metadata.namespace)

            data += lines

    fname = f"{dir}/logs-{pods}pods-{jobs}jobs.txt"
    with open(fname, 'w') as f:
        f.write(data)

    print(f"{fname} is created for all logs")


def getmetrics(pods, jobs, logs=False):
    read_con(pods, jobs)
    read_pl(pods, jobs)
    if logs:
        read_all_log(pods, jobs)


def main():
    argparser = argparse.ArgumentParser(
        prog='Get metrics',
        description='Get metrics from the scheduler and controller',
        epilog=''
    )
    argparser.add_argument('-p', '--pods', type=int,
                           help='number of pods', required=True)
    argparser.add_argument('-j', '--jobs', type=int,
                           help='number of jobs', required=True)
    args = argparser.parse_args()

    getmetrics(args.pods, args.jobs)


if __name__ == "__main__":
    main()

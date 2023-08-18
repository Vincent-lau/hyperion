#!/usr/bin/env python3

import datetime
import json
import os
import argparse

from kubernetes import client, config


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


class MetricsGetter:
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
        "generated jobDemand"
    ]

    def __init__(self) -> None:
        self.dir = self.mkdir()
        self.v1 = None

    @classmethod
    def mkdir(cls) -> str:
        dir = ""
        name = datetime.datetime.now().isoformat()[:-7].replace(':', '-')
        dir = f"measure/data/{name}"
        os.mkdir(dir)
        return dir

    def read_pl(self, pods: int, jobs: int, top: int, metrics: dict):
        ret = self.v1.list_namespaced_pod(namespace='dist-sched', watch=False)
        for i in ret.items:
            if i.metadata.name.startswith("my-scheduler-") or i.metadata.name.startswith("my-controller"):
                # print(f"{i.metadata.name} {i.status.pod_ip}")
                lines = self.v1.read_namespaced_pod_log(
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

                        for sn in self.placement_stats_name:
                            if sn in d:
                                m[sched_name][sn] = d[sn]
        fname = f"{self.dir}/placement-{pods}pods-{jobs}jobs-{top}top.json"

        with open(fname, 'w') as f:
            json.dump(metrics, f)

        print(f"{fname} is created for placement")

    def read_con(self, pods: int, jobs: int, top: int, metrics: dict):
        ret = self.v1.list_namespaced_pod(namespace='dist-sched', watch=False)
        for i in ret.items:
            if i.metadata.name.startswith("my-scheduler-") or i.metadata.name.startswith("my-controller"):
                # print(f"{i.metadata.name} {i.status.pod_ip}")
                lines = self.v1.read_namespaced_pod_log(
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

                        for sn in self.consensus_stats_name:
                            # if sn.endswith('per iter'):
                            #     if not sn in m[sched_name]:
                            #         m[sched_name][sn] = []
                            #     if sn in d:
                            #         m[sched_name][sn].append(d[sn])
                            # else:
                            if sn in d:
                                m[sched_name][sn] = d[sn]

        fname = f"{self.dir}/consensus-{pods}pods-{jobs}jobs-{top}top.json"
        with open(fname, 'w') as f:
            json.dump(metrics, f)

        print(f"{fname} is created for consensus")

    def read_all_log(self, pods: int, jobs: int, top: int):
        ret = self.v1.list_namespaced_pod(namespace='dist-sched', watch=False)
        data = ""
        for i in ret.items:
            if i.metadata.name.startswith("my-scheduler-") or i.metadata.name.startswith("my-controller"):
                # print(f"{i.metadata.name} {i.status.pod_ip}")
                lines = self.v1.read_namespaced_pod_log(
                    name=i.metadata.name, namespace=i.metadata.namespace)

                data += lines

        fname = f"{self.dir}/logs-{pods}pods-{jobs}jobs-{top}top.txt"
        with open(fname, 'w') as f:
            f.write(data)

        print(f"{fname} is created for all logs")

    def getmetrics(self, pods: int, jobs: int, top: int, logs=False):
        # Configs can be set in Configuration class directly or using helper utility
        self.get_k8s()
        metrics = {}
        if logs:
            self.read_all_log(pods, jobs, top)
        self.read_con(pods, jobs, top, metrics)
        self.read_pl(pods, jobs, top, metrics)
        self.reset_k8s()

    def get_k8s(self):
        config.load_kube_config()
        self.v1 = client.CoreV1Api()

    def reset_k8s(self):
        self.v1.api_client.rest_client.pool_manager.clear()
        self.v1.api_client.close()


def main():
    argparser = argparse.ArgumentParser(
        prog='Get metrics',
        description='Get metrics from the scheduler and controller',
        epilog=''
    )
    argparser.add_argument('-p', '--pods', type=int,
                           help='number of scheduler pods', required=True)
    argparser.add_argument('-j', '--jobs', type=int,
                           help='number of jobs', required=True)
    argparser.add_argument('-t', '--topology', type=int,
                           help='topology id', required=True)
    argparser.add_argument('-l', '--logs', action='store_true', default=False,
                           help='get all logs')
    args = argparser.parse_args()

    metrics_getter = MetricsGetter()
    metrics_getter.getmetrics(args.pods, args.jobs, args.topology, args.logs)


if __name__ == "__main__":
    main()

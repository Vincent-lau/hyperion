#! /usr/bin/env python3

import os
import sys
import time
from rich.console import Console
from jinja2 import Environment, FileSystemLoader
from inputimeout import inputimeout, TimeoutOccurred
from contextlib import suppress


def install(num_jobs: int):
    t = 0
    open('deploy/bbox-pod.yaml', 'w').close()
    with open('deploy/google-cluster.csv', 'r') as in_file:
        i = 0
        for line in in_file:
            cpu = round(float(line.split(',')[2]) * 1000, 1)
            if cpu < 1:
                continue
            t = max(t, cpu)
            with open('deploy/templates/bbox-pod-tpl.yaml', 'r') as pod_file:
                pod = pod_file.read()
                name = f'bbox-sleep{cpu}-{i}'
                pod = pod.replace('%CPU%', f'{str(cpu)}m')\
                    .replace('%NAME%', name)\
                    .replace('%SLEEP%', str(cpu))

                with open('deploy/bbox-pod.yaml', 'a') as out_file:
                    out_file.write(pod)
                    out_file.write('---\n')

            i += 1
            if i >= num_jobs:
                break
    os.system('kubectl apply -f deploy/bbox-pod.yaml')
    return t


def uninstall(num_jobs: int):
    with open('deploy/google-cluster.csv', 'r') as in_file:
        i = 0
        for line in in_file:
            cpu = round(float(line.split(',')[2]) * 1000, 1)
            if cpu < 1:
                continue
            with open('deploy/templates/bbox-pod-tpl.yaml', 'r') as pod_file:
                pod = pod_file.read()
                name = f'bbox-sleep{cpu}-{i}'
                pod = pod.replace('%CPU%', f'{str(cpu)}m')\
                    .replace('%NAME%', name)\
                    .replace('%SLEEP%', str(cpu))

                with open('deploy/bbox-pod.yaml', 'a') as out_file:
                    out_file.write(pod)
                    out_file.write('---\n')

                # print(
                #     f'deploying a busybox {name} that will sleep for {cpu} seconds')
                # os.system('kubectl delete -f deploy/bbox-pod.yaml')

            i += 1
            if i >= num_jobs:
                break

    os.system('kubectl delete -f deploy/bbox-pod.yaml')


def render_bbox(num_jobs: int, scheduler_name: str):
    environment = Environment(loader=FileSystemLoader("deploy/templates"))
    template = environment.get_template("bbox-job.yaml.jinja2")
    with open('deploy/bbox-job.yaml', 'w') as out_file:
        for st in [1, 5, 10]:
            content = template.render(
                scheduler_name=scheduler_name,
                num_jobs=num_jobs,
                st=st,
                cpu=st,
            )
            out_file.write(content)
            out_file.write('\n---\n')


def render_pi(num_jobs: int, scheduler_name: str):
    environment = Environment(loader=FileSystemLoader("deploy/templates"))
    template = environment.get_template("pi-job.yaml.jinja2")
    with open('deploy/bbox-job.yaml', 'w') as out_file:
        for d in [100, 500, 5000]:
            content = template.render(
                scheduler_name=scheduler_name,
                num_jobs=num_jobs,
                digits=d,
                cpu=int(d/10),
            )
            out_file.write(content)
            out_file.write('\n---\n')

def render_pod(num_pods: int) -> float:
    environment = Environment(loader=FileSystemLoader("deploy/templates"))
    template = environment.get_template("bbox-pod.yaml.jinja2")
    max_st = 0
    with open('deploy/bbox-pod.yaml', 'w') as out_file:
        with open('deploy/google-cluster.csv', 'r') as usage:
            i = 0
            for line in usage:
                cpu = int(float(line.split(',')[2]) * 1000)
                sleep_time = round(float(line.split(',')[2]) * 1000, 2)
                max_st = max(max_st, sleep_time)
                if cpu < 1:
                    continue
                
                content = template.render(
                    name=f'bbox-sleep{sleep_time}-{i}',
                    cpu=cpu,
                    sleep_time=sleep_time,
                )
                out_file.write(content)
                out_file.write('\n---\n')

                i += 1
                if i >= num_pods:
                    break

    return max_st

def run_jobs():
    console = Console()
    schedulers = ['default-scheduler', 'my-controller']
    scheduler_name = schedulers[1]

    for i in range(1, 2):
        console.print(f'Running {i * 27} jobs', style='green bold')
        render_pi(i * 27 / 3, scheduler_name)
        os.system('kubectl apply -f deploy/bbox-job.yaml')

        wt = 0
        if i < 8:
            wt = 180
        else:
            wt = 240
        console.print(f'sleeping for {wt} seconds', style='red')

        with suppress(TimeoutOccurred):
            inputimeout(prompt=':B', timeout=wt)

        os.system('kubectl delete -f deploy/bbox-job.yaml')
    console.print('done', style='blue')


def run_pods():
    console = Console()
    for i in range(1, 10):
        console.print(f'Running {i} pods', style='green bold')
        wt = render_pod(i) + 120
        os.system("kubectl apply -f deploy/bbox-pod.yaml")

        console.print(f'sleeping for {wt} seconds', style='red bold')
        with suppress(TimeoutOccurred):
            inputimeout(prompt=':D', timeout=wt)

        os.system('kubectl delete -f deploy/bbox-pod.yaml')

def main():
    run_pods()



if __name__ == '__main__':
    main()

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
            with open('deploy/bbox-pod-tpl.yaml', 'r') as pod_file:
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
                # os.system('kubectl apply -f deploy/bbox-pod.yaml')

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
            with open('deploy/bbox-pod-tpl.yaml', 'r') as pod_file:
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


def render_tpl(num_jobs: int, scheduler_name: str):
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


def main():
    console = Console()
    schedulers = ['default-scheduler', 'my-controller']
    scheduler_name = schedulers[1]

    for i in range(7, 11):
        console.print(f'Running {i * 27} jobs', style='green bold')
        render_tpl(i * 27 / 3, scheduler_name)
        os.system('kubectl apply -f deploy/bbox-job.yaml')

        wt = 0
        if i < 8:
            wt = 120
        else:
            wt = 180
        console.print(f'sleeping for {wt} seconds', style='red')

        with suppress(TimeoutOccurred):
            inputimeout(prompt=':D', timeout=wt)

        os.system('kubectl delete -f deploy/bbox-job.yaml')
    console.print('done', style='blue')


# def main():
#     console = Console()
#     for i in range(8* 27, 9 * 27, 27):
#         t = install(i) + 120
#         console.print(f'sleeping for {t} seconds', style='underline red')
#         time.sleep(int(t))
#         uninstall(i)


main()

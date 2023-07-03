#! /usr/bin/env python3


'''
This script is used to run jobs on the cluster.

'''

import math
import os
from contextlib import suppress

from inputimeout import TimeoutOccurred, inputimeout
from jinja2 import Environment, FileSystemLoader
from rich.console import Console
from scipy.optimize import fsolve


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
                pod = pod.replace('%CPU%', f'{str(cpu)}m') \
                    .replace('%NAME%', name) \
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
                pod = pod.replace('%CPU%', f'{str(cpu)}m') \
                    .replace('%NAME%', name) \
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
    with open('deploy/pi-job.yaml', 'w') as out_file:
        for d in [2000, 4000, 6000]:
            content = template.render(
                scheduler_name=scheduler_name,
                num_jobs=int(num_jobs),
                digits=d,
                cpu=int(d / 10),
            )
            out_file.write(content)
            out_file.write('\n---\n')


def render_pod_fft(num_pods: int, scheduler_name: str) -> float:
    environment = Environment(loader=FileSystemLoader("deploy/templates"))
    template = environment.get_template("fft-pod.yaml.jinja2")
    max_st = 0
    with open('deploy/fft-pod.yaml', 'w') as out_file:
        with open('deploy/google-cluster.csv', 'r') as usage:
            i = 0
            for line in usage:
                c = float(line.split(',')[2]) * 1000
                cpu = int(c)
                if cpu < 1:
                    continue
                k = 0.34293475

                def func(x):
                    return k * x * math.log(x) - c * 60 * 10e9

                solution = fsolve(func, 1000000000)
                n = int(solution[0] / 1e5)
                max_st = max(max_st, n)

                content = template.render(
                    scheduler_name=scheduler_name,
                    name=f'fft-{n}-{i}',
                    cpu=cpu,
                    n=n,
                )
                out_file.write(content)
                out_file.write('\n---\n')

                i += 1
                if i >= num_pods:
                    break

    return max_st


def render_pod_fib(num_pods: int, scheduler_name: str) -> float:
    environment = Environment(loader=FileSystemLoader("deploy/templates"))
    template = environment.get_template("fib-pod.yaml.jinja2")
    max_st = 0
    with open('deploy/fib-pod.yaml', 'w') as out_file:
        with open('deploy/google-cluster.csv', 'r') as usage:
            i = 0
            for line in usage:
                cpu = int(float(line.split(',')[2]) * 1000)
                n = int(float(line.split(',')[2]) * 1000 * 30000)
                max_st = max(max_st, n)
                if cpu < 1 or n > 5000000:
                    continue

                content = template.render(
                    scheduler_name=scheduler_name,
                    name=f'fib-{n}th-{i}',
                    cpu=cpu,
                    n=n,
                )
                out_file.write(content)
                out_file.write('\n---\n')

                i += 1
                if i >= num_pods:
                    break

    return max_st


def render_pod_pi(num_pods: int, scheduler_name: str) -> float:
    environment = Environment(loader=FileSystemLoader("deploy/templates"))
    template = environment.get_template("pi-pod.yaml.jinja2")
    max_st = 0
    with open('deploy/pi-pod.yaml', 'w') as out_file:
        with open('deploy/google-cluster.csv', 'r') as usage:
            i = 0
            for line in usage:
                cpu = int(float(line.split(',')[2]) * 1000)
                digits = int(math.pow(cpu, 2 / 3) * 300)
                max_st = max(max_st, digits)
                if cpu < 1:
                    continue

                content = template.render(
                    scheduler_name=scheduler_name,
                    name=f'pi-digits{digits}-{i}',
                    cpu=cpu,
                    digits=digits,
                )
                out_file.write(content)
                out_file.write('\n---\n')

                i += 1
                if i >= num_pods:
                    break

    return max_st


def render_pod_bbox(num_pods: int, scheduler_name: str) -> float:
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
                    scheduler_name=scheduler_name,
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


def run_jobs(scheduler_name: str, job_name: str):
    console = Console()
    job_factor = 36

    for i in range(5, 6):
        console.print(f'Running {i * job_factor} jobs', style='green bold')
        if job_name == 'pi':
            render_pi(i * job_factor / 3, scheduler_name)
        elif job_name == 'bbox':
            render_bbox(i * job_factor / 3, scheduler_name)
        os.system(f'kubectl apply -f deploy/{job_name}-job.yaml')

        wt = 0
        if i < 8:
            wt = 180
        else:
            wt = 240
        console.print(f'sleeping for {wt} seconds', style='red')

        with suppress(TimeoutOccurred):
            inputimeout(prompt=':B', timeout=wt)

        os.system(f'kubectl delete -f deploy/{job_name}-job.yaml')
    console.print('done', style='blue')


def run_pods(scheduler_name: str, pod_name: str):
    console = Console()
    job_factor = 9 * 12
    for i in range(5, 10):
        console.print(
            f'Running {i * job_factor} {pod_name} pods', style='green bold')
        if pod_name == 'bbox':
            render_pod_bbox(i * job_factor, scheduler_name)
        elif pod_name == 'pi':
            render_pod_pi(i * job_factor, scheduler_name)
        elif pod_name == 'fib':
            render_pod_fib(i * job_factor, scheduler_name)
        elif pod_name == 'fft':
            render_pod_fft(i * job_factor, scheduler_name)
        else:
            raise Exception(f'Unknown pod name {pod_name}')

        wt = 180
        os.system(f"kubectl apply -f deploy/{pod_name}-pod.yaml")

        console.print(f'sleeping for {wt} seconds', style='red')
        with suppress(TimeoutOccurred):
            inputimeout(prompt=':D', timeout=wt)

        os.system(f'kubectl delete -f deploy/{pod_name}-pod.yaml')

        with suppress(TimeoutOccurred):
            inputimeout(prompt=f'waiting after deleting', timeout=60)


def main():
    schedulers = ['default-scheduler', 'my-controller']
    scheduler_name = schedulers[0]
    # for _ in range(5):
    # render_pod_fft(30, scheduler_name)
    # run_pods(scheduler_name, 'pi')
    # for _ in range(3):
    #     for sn in schedulers:
    run_pods(scheduler_name, 'fft')
    # run_jobs(scheduler_name, 'pi')


if __name__ == '__main__':
    main()

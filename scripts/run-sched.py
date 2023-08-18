#!/usr/bin/env python3

# Purpose: Run the scheduler with a given number of pods and jobs

from jinja2 import Environment, FileSystemLoader
import pytermgui as ptg
import time
from enum import Enum
import os
import argparse
import getmetrics


class Mode(Enum):
    DEV = 1
    PROD = 2


class Cmd(Enum):
    RUN = 1
    REDEPLOY = 2


# TODO add color to prompt
def macro_time(fmt: str) -> str:
    return time.strftime(fmt)


# ptg.tim.define("!time", macro_time)

# with ptg.WindowManager() as manager:
#     manager.layout.add_slot("Body")
#     manager.add(
#         ptg.Window(
#             "[bold]The current time is:[/]\n\n[!time 75]%c", box="EMPTY")
#     )

def run_ctrler():
    os.system('''
        echo "=====================deploying controller=================="
        kubectl apply -f deploy/my-controller.yaml 
    ''')


def run_sched():
    os.system('''
        kubectl apply -f deploy/my-scheduler.yaml
    
    ''')


def render_sched(pods: int, mode: Mode, top: int):
    environment = Environment(loader=FileSystemLoader("deploy/templates"))
    template = environment.get_template("my-scheduler.yaml.jinja2")
    with open('deploy/my-scheduler.yaml', 'w') as out_file:
        content = template.render(
            replicas=pods,
            mode=mode.name,
            topid=top
        )
        out_file.write(content)


def render_ctrler(pods: int, mode: Mode, top: int, job_factor: int, trials: int):
    environment = Environment(loader=FileSystemLoader("deploy/templates"))
    template = environment.get_template("my-controller.yaml.jinja2")
    trials = 1
    with open('deploy/my-controller.yaml', 'w') as out_file:
        content = template.render(
            replicas=pods,
            mode=mode.name,
            topid=top,
            trials=trials,
            jobfactor=job_factor,
        )
        out_file.write(content)


def remove(wait_time=5):
    os.system('''
            kubectl delete -f deploy/my-controller.yaml --ignore-not-found && \
            kubectl delete -f deploy/my-scheduler.yaml --ignore-not-found
        ''')
    print("waiting for deletion")
    time.sleep(wait_time)


def main():

    parser = argparse.ArgumentParser(
        prog='Deploy scheduler script',
        description='Deploy the controller and scheduler',
        epilog=''
    )

    parser.add_argument('-m', '--make', action='store_true', help='run make')
    parser.add_argument('-d', '--dev', action='store_true',
                        help='run in production mode, default is prod')
    parser.add_argument('-r', '--remove', action='store_true',
                        help='remove previous the controller and scheduler')
    parser.add_argument('-p', '--pods', type=int, nargs='+',
                        help='number of pods to run')
    parser.add_argument('-t', '--topology', type=int,
                        nargs='+', help='topology id')
    parser.add_argument('-j', '--jobs', type=int, nargs='+',
                        help='job factor, the job/scheduler ratio, default to 1')
    parser.add_argument('-w', '--wait', type=int,
                        help='wait time for job completion in seconds')
    parser.add_argument('-l', '--logs', action='store_true', default=False,
                        help='explicitly store logs')
    parser.add_argument('-n', '--run', type=int, nargs='?',
                        default=1, help='how many times to run the experiment')
    args = parser.parse_args()

    if args.remove:
        remove()
        return

    if args.dev:
        print("running in development mode")
        mode = Mode.DEV
    else:
        print("running in production mode")
        mode = Mode.PROD
    if args.make:
        if args.dev:
            os.system("make RACE=1")
        else:
            os.system("make")

    os.system('kubectl config set-context --current --namespace=dist-sched')    

    start_pods = 500
    end_pods = 1000
    if len(args.pods) > 1:
        start_pods = args.pods[0]
        end_pods = args.pods[1]
    elif len(args.pods) == 1:
        start_pods = args.pods[0]
        end_pods = start_pods + 100

    for _ in range(args.run):
        metrics_getter = None
        for pods in range(start_pods, end_pods, 100):
            for jobs in args.jobs if args.jobs else [1]:
                for top in args.topology if args.topology else [2]:
                    print(
                        f"Running with {pods} pods and {jobs * pods} jobs {top} topology")
                    remove()
                    render_ctrler(pods, mode, top, jobs, 1)
                    render_sched(pods, mode, top)
                    run_ctrler()
                    print("waiting for controller to start up")
                    time.sleep(10)
                    run_sched()

                    sleep_time = args.wait if args.wait else max(200, pods)
                    try:
                        for i in range(sleep_time):
                            print(
                                f"Sleeping for {sleep_time - i} seconds while waiting for completion", end='\r', flush=True)
                            time.sleep(1)
                    except KeyboardInterrupt:
                        print("Keyboard interrupt detected, waking up")
                    if mode == Mode.PROD or args.logs:
                        metrics_getter = getmetrics.MetricsGetter(
                        ) if not metrics_getter else metrics_getter
                        metrics_getter.getmetrics(pods, jobs, top, args.logs)


if __name__ == "__main__":
    main()

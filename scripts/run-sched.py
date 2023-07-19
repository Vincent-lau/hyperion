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

        kubectl delete -f deploy/my-controller.yaml --ignore-not-found && \
        kubectl apply -f deploy/my-controller.yaml 
    ''')


def run_sched():
    os.system('''
        kubectl delete -f deploy/my-scheduler.yaml --ignore-not-found && \
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



def main():


    parser = argparse.ArgumentParser(
        prog = 'Deploy scheduler script',
        description='Deploy the controller and scheduler',
        epilog=''
    )

    parser.add_argument('-m', '--make', action='store_true', help='run make')
    parser.add_argument('-d', '--dev', action='store_true', help='run in production mode, default is prod')
    parser.add_argument('-r', '--remove', action='store_true', help='remove previous the controller and scheduler')
    parser.add_argument('-p', '--pods', type=int, nargs = 2, help='number of pods to run')
    args = parser.parse_args()


    mode = Mode.PROD
    if args.dev:
        mode = Mode.DEV
    if args.make:
        if args.dev:
            os.system("make RACE=1")
        else:
            os.system("make")

    if args.remove:
        os.system('''
            kubectl delete -f deploy/my-controller.yaml --ignore-not-found && \
            kubectl delete -f deploy/my-scheduler.yaml --ignore-not-found
        ''')
        return
    
    start_pods = 500
    end_pods = 10000
    if len(args.pods) > 1:
        start_pods = args.pods[0]
        end_pods = args.pods[1]
    elif len(args.pods) == 1:
        start_pods = args.pods[0]
    

    for pods in range(start_pods, end_pods, 100):
        for jobs in range(1, 2):
            for top in range(1, 2):
                print(
                    f"Running with {pods} pods and {jobs * pods} jobs {top} topology")
                render_ctrler(pods, mode, top, 1, 1)
                render_sched(pods, mode, top)
                run_ctrler()
                print("waiting for controller to start up")
                time.sleep(10)
                run_sched()

                sleep_time = max(200, pods)
                print(f"Sleeping for {sleep_time} seconds while waiting for completion")
                try:
                    time.sleep(sleep_time)
                except KeyboardInterrupt:
                    print("Keyboard interrupt detected, waking up")
                getmetrics.getmetrics(pods, jobs)



if __name__ == "__main__":
    main()
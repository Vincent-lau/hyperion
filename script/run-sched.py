#!/usr/bin/env python3

# Purpose: Run the scheduler with a given number of pods and jobs

from jinja2 import Environment, FileSystemLoader
import pytermgui as ptg
import time
from enum import Enum
import os


class Mode(Enum):
    DEV = 1
    PROD = 2


class Cmd(Enum):
    RUN = 1
    REDEPLOY = 2


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

        # compile the protobuf
        protoc --go_out=. --go_opt=paths=source_relative \
        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
            message/message.proto

        go build -o ./bin/ctl cmd/controller/main.go && \
        docker build -t my-ctl -f cmd/controller/Dockerfile . && \
        docker tag my-ctl cuso4/my-ctl && \
        docker push cuso4/my-ctl && \
        kubectl delete -f deploy/my-controller.yaml --ignore-not-found && \
        kubectl apply -f deploy/my-controller.yaml 
    ''')


def run_sched():
    os.system('''
        echo "=====================deploying scheduler=================="
        go build -o ./bin/sched cmd/scheduler/main.go && \
        docker build -t my-sched -f cmd/scheduler/Dockerfile . && \
        docker tag my-sched cuso4/my-sched && \
        docker push cuso4/my-sched:latest && \
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


for pods in range(9, 10):
    for jobs in range(1, 2):
        for top in range(1, 2):
            print(
                f"Running with {pods} pods and {jobs * pods} jobs {top} topology")
            render_ctrler(pods, Mode.DEV, top, 1, 1)
            render_sched(pods, Mode.DEV, top)
            run_ctrler()
            run_sched()

            sleep_time = max(200, pods)
            print(f"Sleeping for {sleep_time} seconds")
            time.sleep(sleep_time)

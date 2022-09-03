#! /usr/bin/env python3

import numpy as np
import os
import sys

data_dir = 'config/data'


def uninstall():
    with open(f'{data_dir}/sample.txt') as f:
        ts = [float(t) for t in f.read().split(',')]
    for i, t in enumerate(ts):
        os.system(f"helm uninstall bbox-sleep{t}-{i}")


def gen_sample(n: int):
    ts = [round(x, 0) for x in np.random.normal(10, 3, n)]
    with open(f'{data_dir}/sample.txt', 'w') as f:
        for i, t in enumerate(ts):
            if i != len(ts) - 1:
                f.write(f'{t},')
            else:
                f.write(f"{t}\n")


def install():
    with open(f'{data_dir}/sample.txt') as f:
        ts = [float(t) for t in f.read().split(',')]
        print(f"installing {len(ts)} bboxs")

    for i, t in enumerate(ts):
        os.system(
            f'helm install bbox-sleep{t}-{i} --set sleep="{t}" ./deploy/bbox')


def main():
    if len(sys.argv) < 2:
        print('Usage: run-bbox.py [install|uninstall|gen-sample]')
        sys.exit(1)

    if sys.argv[1] == 'uninstall':
        uninstall()
    elif sys.argv[1] == 'install':
        install()
    elif sys.argv[1] == 'gen-sample':
        gen_sample(int(sys.argv[2]))
    else:
        print('Usage: run-bbox.py [install|uninstall|gen-sample]')
        sys.exit(1)

main()

#!/usr/bin/python3

from functools import reduce
import sys

fib = lambda n: reduce(lambda x, _: [x[1],x[0]+x[1]], range(n), [0, 1])[0]
n = int(sys.argv[1])
print(fib(n))

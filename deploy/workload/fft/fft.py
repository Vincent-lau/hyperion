#!/usr/bin/env python3

from scipy.fft import fft, ifft
import numpy as np
import sys

n = int(sys.argv[1])
x = np.random.random(n)

# x = np.array([1.0, 2.0, 1.0, -1.0, 1.5])
y = fft(x)


print(y)

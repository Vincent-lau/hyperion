FROM python:3

WORKDIR /usr/src/app
COPY workload/fft/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY workload/fft/fft.py ./

ENTRYPOINT ["./fft.py"]

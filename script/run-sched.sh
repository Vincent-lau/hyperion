#!/bin/zsh

go build -o ./app/main main/main.go && \
docker build -t my-sched . && \
docker tag my-sched cuso4/my-sched && \
docker push cuso4/my-sched && \
kubectl delete -f deploy/my-scheduler.yaml && \
kubectl apply -f deploy/my-scheduler.yaml


#!/bin/zsh

go build -o ./app/main main/main.go && \
docker build -t rand-sched . && \
docker tag rand-sched cuso4/rand-sched && \
docker push cuso4/rand-sched && \
kubectl delete -f deploy/rand-scheduler.yaml && \
kubectl apply -f deploy/rand-scheduler.yaml


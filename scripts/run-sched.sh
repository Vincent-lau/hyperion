#!/bin/zsh


echo "===================deploying scheduler==================="



go build -o ./bin/sched cmd/scheduler/main.go && \
docker build -t my-sched -f cmd/scheduler/Dockerfile . && \
docker tag my-sched cuso4/my-sched && \
docker push cuso4/my-sched:latest && \
kubectl delete -f deploy/my-scheduler-deploy.yaml --ignore-not-found && \
kubectl apply -f deploy/my-scheduler-deploy.yaml

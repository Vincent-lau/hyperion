#!/bin/zsh

echo "=====================deploying controller=================="

go build -race -o ./cmd/controller/ctl cmd/controller/main.go && \
docker build -t my-ctl cmd/controller && \
docker tag my-ctl cuso4/my-ctl && \
docker push cuso4/my-ctl && \
kubectl delete -f deploy/my-controller-pod.yaml && \
kubectl apply -f deploy/my-controller-pod.yaml


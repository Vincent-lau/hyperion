#!/bin/zsh

echo "=====================deploying controller=================="

go build -race -o ./bin/ctl cmd/controller/main.go && \
docker build -t my-ctl -f cmd/controller/Dockerfile . && \
docker tag my-ctl cuso4/my-ctl && \
docker push cuso4/my-ctl && \
kubectl delete -f deploy/my-controller-pod.yaml && \
kubectl apply -f deploy/my-controller-pod.yaml


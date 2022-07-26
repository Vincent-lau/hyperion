#!/bin/zsh

kubectl delete -f deploy/my-controller-pod.yaml
kubectl delete -f deploy/my-scheduler.yaml

kubectl apply -f deploy/my-controller-pod.yaml

sleep 5

kubectl apply -f deploy/my-scheduler.yaml

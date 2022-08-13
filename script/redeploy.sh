#!/bin/zsh

kubectl delete -f deploy/my-controller-pod.yaml --ignore-not-found 
kubectl delete -f deploy/my-scheduler-deploy.yaml --ignore-not-found

kubectl apply -f deploy/my-controller-pod.yaml

sleep 5

kubectl apply -f deploy/my-scheduler-deploy.yaml

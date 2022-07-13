#!/bin/sh


# compile the protobuf
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    message/message.proto


GOOS=linux go build -o ./app/main main/main.go
docker build -t in-cluster .

docker tag in-cluster cuso4/in-cluster
docker push cuso4/in-cluster

kubectl apply -f deploy/nine-pod.yaml
sleep 4
kubectl apply -f deploy/nine-pod-svc.yaml


watch kubectl get pods

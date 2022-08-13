#!/bin/zsh

echo "=====================deploying controller=================="

# compile the protobuf
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    message/message.proto

go build -race -o ./bin/ctl cmd/controller/main.go && \
docker build -t my-ctl -f cmd/controller/Dockerfile . && \
docker tag my-ctl cuso4/my-ctl && \
docker push cuso4/my-ctl && \
kubectl delete -f deploy/my-controller-pod.yaml --ignore-not-found && \
kubectl apply -f deploy/my-controller-pod.yaml


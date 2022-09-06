#!/bin/zsh

echo "=====================deploying random scheduler=================="

# compile the protobuf
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    message/message.proto

go build -race -o ./bin/rand cmd/rand_sched/main.go && \
docker build -t my-rand -f cmd/rand_sched/Dockerfile . && \
docker tag my-rand cuso4/my-rand && \
docker push cuso4/my-rand && \
kubectl delete -f deploy/my-rand-pod.yaml --ignore-not-found && \
kubectl apply -f deploy/my-rand-pod.yaml


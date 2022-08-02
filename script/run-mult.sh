#!/bin/zsh


for run in {2..3}
do
  echo "-------------------run $run-------------------"
  for i in 50 100 150 200
  do
    echo "Running with $i pods"

    sed s/%REPLICAS%/$i/g deploy/my-scheduler-temp.yaml > deploy/my-scheduler-deploy.yaml 
    sed s/%REPLICAS%/$i/g deploy/my-controller-temp.yaml > deploy/my-controller-pod.yaml 
  
    
    # make does not work, we need to wait for the controller a bit 
    ./script/run.sh

    echo "waiting for algorithm to complete..."
    sleep $i

    echo "writing results to data-${i}pods.json"
    ./script/getmetrics.py > "measure/data/data-${i}pods-run${run}.json"

  done
done

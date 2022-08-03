#!/bin/zsh


for run in 0
do
  gum style --foreground 212 --border-foreground 102 --border normal \
            --align center --width 50 --margin "1 2" --padding "2 4" \
            "run $run"
  for i in 9
  do
    gum style --foreground 103 --border-foreground 212 --border double \
              --align center --width 50 --margin "1 2" --padding "2 4" \
              "Running with $i pods"

    sed s/%REPLICAS%/$i/g deploy/my-scheduler-temp.yaml > deploy/my-scheduler-deploy.yaml 
    sed s/%REPLICAS%/$i/g deploy/my-controller-temp.yaml > deploy/my-controller-pod.yaml 
  
    # make does not work, we need to wait for the controller a bit 
    CMD=$(gum choose "run.sh" "redeploy.sh" --limit=1 || echo "run.sh")
    # CMD="run.sh"
    ./script/$CMD

    # date
    # gum spin -s line --title "waiting for algorithm to complete" sleep 200
    # date

    # gum style --foreground 212 "writing results to data-${i}pods-run${run}.json"
    # ./script/getmetrics.py > "measure/data/data-${i}pods-run${run}.json"

  done
done

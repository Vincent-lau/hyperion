#! /bin/zsh


for i in {1..10}
do
    NUM=$(expr $i \* 27) 
    gum style --foreground 103 --border-foreground 212 --border double \
              --align center --width 50 --margin "1 2" --padding "2 4" \
              "Running with $NUM completions"

    st=$(expr $i \* 5) 
    sed "s/%NUM_JOBS%/$NUM/g" deploy/bbox-job-tpl.yaml > deploy/bbox-job.yaml 
    kubectl apply -f deploy/bbox-job.yaml
    gum spin -s line --title "waiting for algorithm to complete, sleeping for 120 seconds" sleep 120
    kubectl delete -f deploy/bbox-job.yaml

done

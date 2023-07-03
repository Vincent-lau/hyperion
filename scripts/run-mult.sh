#!/bin/zsh

MODE=$(gum choose "DEV" "PROD" --limit=1 || echo "DEV")
CMD=$(gum choose "run.sh" "redeploy.sh" --limit=1 || echo "run.sh")


for i in {100..900..100}
# $(gum choose --no-limit 9 100 200 300 400 500 600 700 800 900)
do
  for j in {1..1..1}
  # $(gum choose --no-limit 1 2 3 4 5 10 12 0.5)
  do
    for k in {1..1..1}
    do
    gum style --foreground 103 --border-foreground 212 --border double \
              --align center --width 50 --margin "1 2" --padding "2 4" \
              "Running with $i pods and $j * $i jobs $k topology"

    case $MODE in
      DEV)
        TRIAL=1
        ;;
      PROD)
        TRIAL=1
        ;;
    esac

    sed "s/%REPLICAS%/$i/g; s/%MODE%/$MODE/g; s/%JOBS%/$j/g; s/%TRIALS%/$TRIAL/g; s/%TOPID%/$k/g" deploy/my-controller-temp.yaml > deploy/my-controller-pod.yaml 
    sed "s/%REPLICAS%/$i/g; s/%MODE%/$MODE/g; s/%TOPID%/$k/g" deploy/my-scheduler-temp.yaml > deploy/my-scheduler-deploy.yaml 

    # make does not work, we need to wait for the controller a bit 

    # CMD="run.sh"
    ./script/$CMD

    mt=200
    st=$(( i > mt ? i : mt ))
    gum spin -s line --title "waiting for algorithm to complete, sleeping for $st seconds" \
      sleep $st

    case $MODE in
      "PROD")
        # ./script/gettrace.py
        gum style --foreground 214 "writing results to data-${i}pods-${j}jobs.json"
        ./script/getmetrics.py ${i} ${j}
        ;;
      *)
        ;;
    esac
  done
done
done
package controller

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	schedulerName = "my-controller"
)



func (ctl *Controller) findNodes() {
	// TODO add informer to get the list of nodes
	nodes, _ := ctl.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s!=%s", "node-role.kubernetes.io/control-plane", ""),
	})
	nMap := make(map[string]*v1.Node)

	for _, node := range nodes.Items {
		nMap[node.Name] = node.DeepCopy()
	}

	ctl.nodeMap = nMap
}

func (ctl *Controller) jobsFromPodQueue(podChan chan *v1.Pod) {
	for {
		// get pods in all the namespaces by omitting namespace
		// Or specify namespace to get pods in particular namespace

		pods, err := ctl.clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Panic("error getting pods")
		}

		log.WithFields(log.Fields{
			"number of pods": len(pods.Items),
		}).Info("There are pods in the cluster")

		watch, err := ctl.clientset.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.schedulerName=%s,spec.nodeName=", schedulerName),
		})
		if err != nil {
			panic(err.Error())
		}

		for event := range watch.ResultChan() {
			if event.Type != "ADDED" {
				continue
			}
			p := event.Object.(*v1.Pod)
			podChan <- p

			log.WithFields(log.Fields{
				"pod name":      p.Name,
				"pod namespace": p.Namespace,
			}).Debug("found a pod to schedule")
		}

	}
}

func getJobDemand(pod *v1.Pod) float64 {

	var tot int64 = 0
	for _, c := range pod.Spec.Containers {
		cpu := c.Resources.Requests.Cpu().MilliValue()
		tot += cpu
	}
	log.WithFields(log.Fields{
		"pod name": pod.Name,
		"cpu":      tot,
	}).Debug("total cpu requests for pod")

	return float64(tot)
}

func (ctl *Controller) placePodToNode(n string, j int) error {
	node := ctl.nodeMap[n]
	pod := ctl.jobPod[j]

	for {
		err := ctl.clientset.CoreV1().Pods(pod.Namespace).Bind(context.TODO(), &v1.Binding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
			Target: v1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Node",
				Name:       node.Name,
			},
		}, metav1.CreateOptions{})

		if err != nil {
			log.WithFields(log.Fields{
				"pod name":  pod.Name,
				"node name": node.Name,
				"error":     err,
			}).Warn("error binding pod to node")
		} else {
			log.WithFields(log.Fields{
				"pod name":  pod.Name,
				"node name": node.Name,
			}).Debug("successfully binding pod to node")
			break
		}
	}

	ctl.placed <- j

	timestamp := time.Now().UTC()
	ctl.clientset.CoreV1().Events(pod.Namespace).Create(context.TODO(), &v1.Event{
		Count:          1,
		Message:        "binding pod to node",
		Reason:         "Scheduled",
		LastTimestamp:  metav1.NewTime(timestamp),
		FirstTimestamp: metav1.NewTime(timestamp),
		Type:           "Normal",
		Source: v1.EventSource{
			Component: schedulerName,
		},
		InvolvedObject: v1.ObjectReference{
			Kind:      "Pod",
			Name:      pod.Name,
			Namespace: pod.Namespace,
			UID:       pod.UID,
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: pod.Name + "-",
		},
	}, metav1.CreateOptions{})

	return nil

}



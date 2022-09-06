package rand_sched

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const schedulerName = "rand-scheduler"

func findNodes(nodes *v1.NodeList, clientset *kubernetes.Clientset) *v1.Node {
	return &nodes.Items[rand.Intn(len(nodes.Items))]
}

func Schedule() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	watch, _ := clientset.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.schedulerName=%s,spec.nodeName=", schedulerName),
	})

	nodes, _ := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})

	for event := range watch.ResultChan() {
		if event.Type != "ADDED" {
			continue
		}
		p := event.Object.(*v1.Pod)
		log.WithFields(log.Fields{
			"namespace": p.Namespace,
			"pod":       p.Name,
		}).Debug("found a pod to schedule")

		randomNode := findNodes(nodes, clientset)

		clientset.CoreV1().Pods(p.Namespace).Bind(context.TODO(), &v1.Binding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.Name,
				Namespace: p.Namespace,
			},
			Target: v1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Node",
				Name:       randomNode.Name,
			},
		}, metav1.CreateOptions{})

		timestamp := time.Now().UTC()
		clientset.CoreV1().Events(p.Namespace).Create(context.TODO(), &v1.Event{
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
				Name:      p.Name,
				Namespace: p.Namespace,
				UID:       p.UID,
			},
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: p.Name + "-",
			},
		}, metav1.CreateOptions{})

	}

}

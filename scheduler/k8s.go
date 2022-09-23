package scheduler

import (
	"context"
	"fmt"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

func MyNode(hostname string) (*v1.Node, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	pods, err := clientset.CoreV1().Pods("dist-sched").List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", hostname),
	})
	if err != nil {
		panic(err.Error())
	}

	n, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", "kubernetes.io/hostname", pods.Items[0].Spec.NodeName),
	})

	return &n.Items[0], err
}

func (sched *Scheduler) MyCpu() (int64, int64) {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	mc, err := metrics.NewForConfig(config)
	if err != nil {
		log.Panic(err)
	}

	nm, err := mc.MetricsV1beta1().NodeMetricses().Get(context.TODO(), sched.onNode.Name, metav1.GetOptions{})
	if err != nil {
		log.Panic(err)
	}

	used := nm.Usage.Cpu().MilliValue()
	total := sched.onNode.Status.Capacity.Cpu().MilliValue()

	return used, total
}

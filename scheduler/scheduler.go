package scheduler

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
	"sync"

	pb "example/dist_sched/message"

	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Scheduler struct {
	mu       sync.Mutex
	hostname string
	conns    map[string]*grpc.ClientConn // connected peers
	stubs    map[string]pb.MaxConsensusClient
	curMax   int
	done     bool

	pb.UnimplementedGreeterServer
	pb.UnimplementedMaxConsensusServer
}

const (
	schedulerName = "my-scheduler"
)

func New(n int) *Scheduler {

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Could not get hostname: %v\n", err)
	}

	go MyServer()

	conns, stubs := MyClient()

	scheduler := &Scheduler{
		hostname: hostname,
		conns:    conns,
		stubs:    stubs,
		curMax:   n,
	}

	return scheduler

}

func (sched *Scheduler) Done() bool {
	return sched.done
}

func (sched *Scheduler) GetCurMax() int {
	return sched.curMax
}

func findNode(clientset *kubernetes.Clientset) (*v1.Node, error) {
	// TODO add informer to get the list of nodes
	nodes, _ := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	return &nodes.Items[rand.Intn(len(nodes.Items))], nil

}

func (sched *Scheduler) PlacePod() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	for {
		// get pods in all the namespaces by omitting namespace
		// Or specify namespace to get pods in particular namespace

		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

		watch, err := clientset.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{
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
			fmt.Println("found a pod to schedule:", p.Namespace, "/", p.Name)

			toBind, err := findNode(clientset)
			if err != nil {
				panic(err.Error())
			}

			clientset.CoreV1().Pods(p.Namespace).Bind(context.TODO(), &v1.Binding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      p.Name,
					Namespace: p.Namespace,
				},
				Target: v1.ObjectReference{
					APIVersion: "v1",
					Kind:       "Node",
					Name:       toBind.Name,
				},
			}, metav1.CreateOptions{})

			log.Printf("binding pod %v to node %v\n", p.Name, toBind.Name)

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

		time.Sleep(10 * time.Second)
	}

}

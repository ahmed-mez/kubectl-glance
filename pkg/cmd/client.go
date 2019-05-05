package cmd

import (
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const namespaces = "namespaces"
const pods = "pods"
const services = "services"
const configs = "config maps"
const pvcs = "persistent volume claims"
const sas = "service accounts"
const secrets = "secrets"
const endpoints = "endpoints"
const daemonsets = "daemonsets"
const deploys = "deployments"
const replicasets = "replica sets"
const statefulsets = "stateful sets"
const jobs = "jobs"

const timeout = time.Duration(120) // TODO: Make the timeout value configurable

func (o *globalSettings) InitClient() {
	restConfig, err := o.configFlags.ToRESTConfig()
	if err != nil {
		panic(err)
	}
	c := kubernetes.NewForConfigOrDie(restConfig)
	rawKubeConfig := o.configFlags.ToRawKubeConfigLoader()
	ns, _, _ := rawKubeConfig.Namespace()
	o.namespace = ns
	o.client = c
	o.restConfig = restConfig
	o.timeout = timeout
}

// GeNodeForPod gets the node of a pod
func (o *globalSettings) GeNodeForPod(podName string) (string, error) {
	pod, err := o.client.CoreV1().Pods(o.namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("got an error while retrieving pod %s: %s", podName, err)
	}
	return pod.Spec.NodeName, nil
}

func (o *globalSettings) GetNamespacedRessources() (map[string]int, error) {
	namespacedResources := make(map[string]int)
	opts := metav1.ListOptions{}
	ns, err := o.client.CoreV1().Namespaces().List(opts)
	if err != nil {
		return namespacedResources, fmt.Errorf("got an error while getting namespaces: %s", err)
	}

	namespacesCount := len(ns.Items)
	namespacedResources[namespaces] = namespacesCount
	namespacedResources[pods] = 0
	namespacedResources[services] = 0
	namespacedResources[configs] = 0
	namespacedResources[pvcs] = 0
	namespacedResources[secrets] = 0
	namespacedResources[sas] = 0
	namespacedResources[endpoints] = 0
	namespacedResources[daemonsets] = 0
	namespacedResources[deploys] = 0
	namespacedResources[replicasets] = 0
	namespacedResources[statefulsets] = 0
	namespacedResources[jobs] = 0

	o.chans = initBufferedChans(namespacesCount)
	ticker := namespacesCount
	for _, n := range ns.Items {
		go o.GetRessourcesByNamespace(n.Name)
	}
	for {
		select {
		case p := <-o.chans.pods:
			namespacedResources[pods] += p
		case svc := <-o.chans.services:
			namespacedResources[services] += svc
		case c := <-o.chans.configs:
			namespacedResources[configs] += c
		case pvc := <-o.chans.pvcs:
			namespacedResources[pvcs] += pvc
		case sec := <-o.chans.secrets:
			namespacedResources[secrets] += sec
		case sa := <-o.chans.sas:
			namespacedResources[sas] += sa
		case e := <-o.chans.endpoints:
			namespacedResources[endpoints] += e
		case ds := <-o.chans.daemonsets:
			namespacedResources[daemonsets] += ds
		case dep := <-o.chans.deploys:
			namespacedResources[deploys] += dep
		case rs := <-o.chans.replicasets:
			namespacedResources[replicasets] += rs
		case sts := <-o.chans.statefulsets:
			namespacedResources[statefulsets] += sts
		case j := <-o.chans.jobs:
			namespacedResources[jobs] += j
		case <-o.chans.done:
			ticker--
			if ticker == 0 {
				return namespacedResources, nil
			}
		case <-time.After(o.timeout * time.Second):
			return nil, errors.New("Timeout")
		}
	}
}

func (o *globalSettings) GetPersistentVolumes() (int, error) {
	opts := metav1.ListOptions{}
	pv, err := o.client.CoreV1().PersistentVolumes().List(opts)
	if err != nil {
		return 0, fmt.Errorf("got an error while getting pv: %s", err)
	}
	return len(pv.Items), nil
}

func (o *globalSettings) GetNodes() (int, int, string, string, error) {
	opts := metav1.ListOptions{}
	no, err := o.client.CoreV1().Nodes().List(opts)
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("got an error while getting namespaces: %s", err)
	}
	unschedulable := 0
	cpuAllocatable, _ := resource.ParseQuantity("0")
	memAllocatable, _ := resource.ParseQuantity("0")
	for _, n := range no.Items {
		if n.Spec.Unschedulable {
			unschedulable++
		}
		cpuAllocatable.Add(n.Status.Allocatable[corev1.ResourceName("cpu")])
		memAllocatable.Add(n.Status.Capacity[corev1.ResourceName("memory")])
	}
	return len(no.Items), unschedulable, cpuAllocatable.String(), memAllocatable.String(), nil
}

func (o *globalSettings) GetRessourcesByNamespace(namespace string) {
	defer func() {
		o.chans.done <- 1
	}()
	opts := metav1.ListOptions{}
	p, err := o.client.CoreV1().Pods(namespace).List(opts)
	if err == nil {
		o.chans.pods <- len(p.Items)
	}
	svc, err := o.client.CoreV1().Services(namespace).List(opts)
	if err == nil {
		o.chans.services <- len(svc.Items)
	}
	c, err := o.client.CoreV1().ConfigMaps(namespace).List(opts)
	if err == nil {
		o.chans.configs <- len(c.Items)
	}
	sec, err := o.client.CoreV1().Secrets(namespace).List(opts)
	if err == nil {
		o.chans.secrets <- len(sec.Items)
	}
	sa, err := o.client.CoreV1().ServiceAccounts(namespace).List(opts)
	if err == nil {
		o.chans.sas <- len(sa.Items)
	}
	e, err := o.client.CoreV1().Endpoints(namespace).List(opts)
	if err == nil {
		o.chans.endpoints <- len(e.Items)
	}
	pvc, err := o.client.CoreV1().PersistentVolumeClaims(namespace).List(opts)
	if err == nil {
		o.chans.pvcs <- len(pvc.Items)
	}
	ds, err := o.client.AppsV1().DaemonSets(namespace).List(opts)
	if err == nil {
		o.chans.daemonsets <- len(ds.Items)
	}
	dep, err := o.client.AppsV1().Deployments(namespace).List(opts)
	if err == nil {
		o.chans.deploys <- len(dep.Items)
	}
	rs, err := o.client.AppsV1().ReplicaSets(namespace).List(opts)
	if err == nil {
		o.chans.replicasets <- len(rs.Items)
	}
	sts, err := o.client.AppsV1().StatefulSets(namespace).List(opts)
	if err == nil {
		o.chans.statefulsets <- len(sts.Items)
	}
	j, err := o.client.BatchV1().Jobs(namespace).List(opts)
	if err == nil {
		o.chans.jobs <- len(j.Items)
	}
}

func initBufferedChans(buffSize int) *channels {
	return &channels{
		pods:         make(chan int, buffSize),
		services:     make(chan int, buffSize),
		configs:      make(chan int, buffSize),
		pvcs:         make(chan int, buffSize),
		sas:          make(chan int, buffSize),
		secrets:      make(chan int, buffSize),
		endpoints:    make(chan int, buffSize),
		daemonsets:   make(chan int, buffSize),
		deploys:      make(chan int, buffSize),
		replicasets:  make(chan int, buffSize),
		statefulsets: make(chan int, buffSize),
		jobs:         make(chan int, buffSize),
		done:         make(chan int, buffSize),
	}
}

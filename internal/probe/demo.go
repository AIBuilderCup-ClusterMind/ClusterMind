package probe

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apimachineryyaml "k8s.io/apimachinery/pkg/util/yaml"
	sigsyaml "sigs.k8s.io/yaml"
)

// Demo scenario names accepted by the DemoCollector.
const (
	// DemoScenarioHealthy serves the virtual cluster with every workload ready.
	DemoScenarioHealthy = "healthy"
	// DemoScenarioFailure serves the virtual cluster with payment-deployment
	// broken (ImagePullBackOff) plus Warning events for the incident timeline.
	DemoScenarioFailure = "failure"
)

// fixed pod-template-hash stamped on synthesized Pods of every deployment.
const demoPodTemplateHash = "6f7c9d8b4"

// DemoMode interprets the CLUSTERMIND_DEMO flag value.
// enabled=false means demo mode is off (empty/0/false/off/no);
// scenario is healthy or failure otherwise.
func DemoMode(v string) (scenario string, enabled bool, err error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "0", "false", "off", "no":
		return "", false, nil
	case "1", "true", "on", "yes", "healthy":
		return DemoScenarioHealthy, true, nil
	case "failure", "fail", "incident", "broken":
		return DemoScenarioFailure, true, nil
	default:
		return "", false, fmt.Errorf("unknown CLUSTERMIND_DEMO value %q (use 1/healthy or failure)", v)
	}
}

// DemoCollector serves a virtual cluster snapshot parsed from the embedded
// demo manifest instead of a live API server. It implements Collector and is
// a drop-in replacement for ClusterCollector: graph building, impact
// analysis, incident derivation and the web UI all consume the same
// ClusterSnapshot.
type DemoCollector struct {
	manifest  []byte
	namespace string // empty = all namespaces
	scenario  string // DemoScenarioHealthy | DemoScenarioFailure
	// createdAt is stamped on every object so resource ages render stable.
	createdAt time.Time
}

// NewDemoCollector parses and validates the demo manifest up front so a bad
// dataset fails at startup rather than on the first request.
func NewDemoCollector(manifest []byte, namespace, scenario string) (*DemoCollector, error) {
	if len(manifest) == 0 {
		return nil, fmt.Errorf("demo manifest is empty")
	}
	if scenario == "" {
		scenario = DemoScenarioHealthy
	}
	if scenario != DemoScenarioHealthy && scenario != DemoScenarioFailure {
		return nil, fmt.Errorf("unknown demo scenario %q (want healthy|failure)", scenario)
	}
	if _, err := parseDemoManifest(manifest); err != nil {
		return nil, fmt.Errorf("parse demo manifest: %w", err)
	}
	return &DemoCollector{
		manifest:  manifest,
		namespace: namespace,
		scenario:  scenario,
		createdAt: time.Now().UTC().Add(-2 * time.Hour),
	}, nil
}

// Namespace returns the namespace filter (empty string = all namespaces).
func (d *DemoCollector) Namespace() string { return d.namespace }

// Collect rebuilds the virtual snapshot: parses the manifest, applies the
// scenario (workload status), synthesizes Pods and Events, and filters by
// namespace. Safe to call concurrently.
func (d *DemoCollector) Collect(_ context.Context) (*ClusterSnapshot, error) {
	objs, err := parseDemoManifest(d.manifest)
	if err != nil {
		return nil, fmt.Errorf("parse demo manifest: %w", err)
	}
	objs.filter(d.namespace)

	now := time.Now().UTC()
	objs.stampCreatedAt(d.createdAt)
	pods, events := objs.applyScenario(d.scenario, now)
	objs.pods = pods
	objs.events = append(objs.events, events...)

	return &ClusterSnapshot{
		CapturedAt:      now,
		Namespaces:      objs.namespaces,
		Ingresses:       objs.ingresses,
		Services:        objs.services,
		Deployments:     objs.deployments,
		StatefulSets:    objs.statefulsets,
		DaemonSets:      objs.daemonsets,
		ConfigMaps:      objs.configmaps,
		HPAs:            objs.hpas,
		NetworkPolicies: objs.netpols,
		Pods:            objs.pods,
		Events:          objs.events,
	}, nil
}

// demoObjects holds every resource kind the manifest may contain.
type demoObjects struct {
	namespaces   []corev1.Namespace
	ingresses    []networkingv1.Ingress
	services     []corev1.Service
	deployments  []appsv1.Deployment
	statefulsets []appsv1.StatefulSet
	daemonsets   []appsv1.DaemonSet
	configmaps   []corev1.ConfigMap
	hpas         []autoscalingv2.HorizontalPodAutoscaler
	netpols      []networkingv1.NetworkPolicy
	pods         []corev1.Pod
	events       []corev1.Event
}

// parseDemoManifest splits a multi-document YAML manifest and decodes each
// document into its typed object. Unknown kinds are skipped silently so the
// manifest can carry annotations or future resources.
func parseDemoManifest(manifest []byte) (*demoObjects, error) {
	objs := &demoObjects{}
	reader := apimachineryyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(manifest)))
	for {
		doc, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read yaml document: %w", err)
		}
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}
		var tm metav1.TypeMeta
		if err := sigsyaml.Unmarshal(doc, &tm); err != nil {
			return nil, fmt.Errorf("decode type metadata: %w", err)
		}
		if tm.Kind == "" {
			continue
		}
		switch tm.Kind {
		case "Namespace":
			var o corev1.Namespace
			if err := sigsyaml.Unmarshal(doc, &o); err != nil {
				return nil, decodeErr(tm.Kind, err)
			}
			objs.namespaces = append(objs.namespaces, o)
		case "Ingress":
			var o networkingv1.Ingress
			if err := sigsyaml.Unmarshal(doc, &o); err != nil {
				return nil, decodeErr(tm.Kind, err)
			}
			objs.ingresses = append(objs.ingresses, o)
		case "Service":
			var o corev1.Service
			if err := sigsyaml.Unmarshal(doc, &o); err != nil {
				return nil, decodeErr(tm.Kind, err)
			}
			objs.services = append(objs.services, o)
		case "Deployment":
			var o appsv1.Deployment
			if err := sigsyaml.Unmarshal(doc, &o); err != nil {
				return nil, decodeErr(tm.Kind, err)
			}
			objs.deployments = append(objs.deployments, o)
		case "StatefulSet":
			var o appsv1.StatefulSet
			if err := sigsyaml.Unmarshal(doc, &o); err != nil {
				return nil, decodeErr(tm.Kind, err)
			}
			objs.statefulsets = append(objs.statefulsets, o)
		case "DaemonSet":
			var o appsv1.DaemonSet
			if err := sigsyaml.Unmarshal(doc, &o); err != nil {
				return nil, decodeErr(tm.Kind, err)
			}
			objs.daemonsets = append(objs.daemonsets, o)
		case "ConfigMap":
			var o corev1.ConfigMap
			if err := sigsyaml.Unmarshal(doc, &o); err != nil {
				return nil, decodeErr(tm.Kind, err)
			}
			objs.configmaps = append(objs.configmaps, o)
		case "HorizontalPodAutoscaler":
			var o autoscalingv2.HorizontalPodAutoscaler
			if err := sigsyaml.Unmarshal(doc, &o); err != nil {
				return nil, decodeErr(tm.Kind, err)
			}
			objs.hpas = append(objs.hpas, o)
		case "NetworkPolicy":
			var o networkingv1.NetworkPolicy
			if err := sigsyaml.Unmarshal(doc, &o); err != nil {
				return nil, decodeErr(tm.Kind, err)
			}
			objs.netpols = append(objs.netpols, o)
		default:
			// Pod / Event / Secret documents in the manifest are ignored;
			// Pods and Events are synthesized per scenario instead.
			continue
		}
	}
	return objs, nil
}

func decodeErr(kind string, err error) error {
	return fmt.Errorf("decode %s: %w", kind, err)
}

// filter keeps only objects inside the namespace scope. Empty scope keeps
// everything (matches ClusterCollector's "all namespaces" behavior).
func (o *demoObjects) filter(namespace string) {
	if namespace == "" || namespace == "all" {
		return
	}
	keepNS := func(ns string) bool { return ns == namespace }
	kept := o.namespaces[:0]
	for _, n := range o.namespaces {
		if n.Name == namespace {
			kept = append(kept, n)
		}
	}
	o.namespaces = kept

	keptSvc := o.services[:0]
	for _, s := range o.services {
		if keepNS(s.Namespace) {
			keptSvc = append(keptSvc, s)
		}
	}
	o.services = keptSvc

	keptIng := o.ingresses[:0]
	for _, i := range o.ingresses {
		if keepNS(i.Namespace) {
			keptIng = append(keptIng, i)
		}
	}
	o.ingresses = keptIng

	keptDep := o.deployments[:0]
	for _, d := range o.deployments {
		if keepNS(d.Namespace) {
			keptDep = append(keptDep, d)
		}
	}
	o.deployments = keptDep

	keptSTS := o.statefulsets[:0]
	for _, s := range o.statefulsets {
		if keepNS(s.Namespace) {
			keptSTS = append(keptSTS, s)
		}
	}
	o.statefulsets = keptSTS

	keptDS := o.daemonsets[:0]
	for _, d := range o.daemonsets {
		if keepNS(d.Namespace) {
			keptDS = append(keptDS, d)
		}
	}
	o.daemonsets = keptDS

	keptCM := o.configmaps[:0]
	for _, c := range o.configmaps {
		if keepNS(c.Namespace) {
			keptCM = append(keptCM, c)
		}
	}
	o.configmaps = keptCM

	keptHPA := o.hpas[:0]
	for _, h := range o.hpas {
		if keepNS(h.Namespace) {
			keptHPA = append(keptHPA, h)
		}
	}
	o.hpas = keptHPA

	keptNP := o.netpols[:0]
	for _, n := range o.netpols {
		if keepNS(n.Namespace) {
			keptNP = append(keptNP, n)
		}
	}
	o.netpols = keptNP
}

// stampCreatedAt assigns the same creation time to every object so ages
// render as "2h" consistently across the UI.
func (o *demoObjects) stampCreatedAt(t time.Time) {
	mt := metav1.NewTime(t)
	for i := range o.namespaces {
		o.namespaces[i].CreationTimestamp = mt
	}
	for i := range o.ingresses {
		o.ingresses[i].CreationTimestamp = mt
	}
	for i := range o.services {
		o.services[i].CreationTimestamp = mt
	}
	for i := range o.deployments {
		o.deployments[i].CreationTimestamp = mt
	}
	for i := range o.statefulsets {
		o.statefulsets[i].CreationTimestamp = mt
	}
	for i := range o.daemonsets {
		o.daemonsets[i].CreationTimestamp = mt
	}
	for i := range o.configmaps {
		o.configmaps[i].CreationTimestamp = mt
	}
	for i := range o.hpas {
		o.hpas[i].CreationTimestamp = mt
	}
	for i := range o.netpols {
		o.netpols[i].CreationTimestamp = mt
	}
}

// applyScenario fills Deployment status, synthesizes Pods for every
// Deployment and returns scenario-specific Events.
// healthy: all workloads ready, Pods running, Normal change events only.
// failure: payment-deployment has ImagePullBackOff (0/1 ready) with Warning
// events that BuildTwinIncidents turns into a sev1 incident timeline.
func (o *demoObjects) applyScenario(scenario string, now time.Time) ([]corev1.Pod, []corev1.Event) {
	var pods []corev1.Pod
	events := demoChangeEvents(o.deployments, now)

	for i := range o.deployments {
		dep := &o.deployments[i]
		desired := int32(1)
		if dep.Spec.Replicas != nil {
			desired = *dep.Spec.Replicas
		}
		failing := scenario == DemoScenarioFailure && dep.Namespace == "demo" && dep.Name == "payment-deployment"

		if failing {
			// Simulate `kubectl set image ...=nginx:does-not-exist`.
			for c := range dep.Spec.Template.Spec.Containers {
				dep.Spec.Template.Spec.Containers[c].Image = "nginx:does-not-exist"
			}
			lt := metav1.NewTime(now.Add(-10 * time.Minute))
			dep.Status = appsv1.DeploymentStatus{
				ObservedGeneration:  1,
				Replicas:            desired,
				UpdatedReplicas:     desired,
				ReadyReplicas:       0,
				AvailableReplicas:   0,
				UnavailableReplicas: desired,
				Conditions: []appsv1.DeploymentCondition{{
					Type:               appsv1.DeploymentAvailable,
					Status:             corev1.ConditionFalse,
					Reason:             "MinimumReplicasUnavailable",
					Message:            "Deployment does not have minimum availability.",
					LastTransitionTime: lt,
				}},
			}
		} else {
			lt := metav1.NewTime(now.Add(-90 * time.Minute))
			dep.Status = appsv1.DeploymentStatus{
				ObservedGeneration:  1,
				Replicas:            desired,
				UpdatedReplicas:     desired,
				ReadyReplicas:       desired,
				AvailableReplicas:   desired,
				UnavailableReplicas: 0,
				Conditions: []appsv1.DeploymentCondition{
					{
						Type:               appsv1.DeploymentAvailable,
						Status:             corev1.ConditionTrue,
						Reason:             "MinimumReplicasAvailable",
						Message:            "Deployment has minimum availability.",
						LastTransitionTime: lt,
					},
					{
						Type:               appsv1.DeploymentProgressing,
						Status:             corev1.ConditionTrue,
						Reason:             "NewReplicaSetAvailable",
						Message:            fmt.Sprintf("ReplicaSet %s-%s has successfully progressed.", dep.Name, demoPodTemplateHash),
						LastTransitionTime: lt,
					},
				},
			}
		}
		pods = append(pods, demoPods(dep, desired, failing, now)...)
	}

	if scenario == DemoScenarioFailure {
		events = append(events, demoFailureEvents(now)...)
	}
	return pods, events
}

// demoPods synthesizes one Pod per desired replica of the deployment.
func demoPods(dep *appsv1.Deployment, desired int32, failing bool, now time.Time) []corev1.Pod {
	created := now.UTC().Add(-2 * time.Hour)
	labels := map[string]string{}
	for k, v := range dep.Spec.Template.Labels {
		labels[k] = v
	}
	labels["pod-template-hash"] = demoPodTemplateHash

	containers := dep.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		containers = []corev1.Container{{Name: dep.Name, Image: "registry.k8s.io/pause:3.9"}}
	}

	pods := make([]corev1.Pod, 0, desired)
	for i := int32(0); i < desired; i++ {
		name := fmt.Sprintf("%s-%s", dep.Name, demoPodTemplateHash)
		if i > 0 {
			name = fmt.Sprintf("%s-%s-%d", dep.Name, demoPodTemplateHash, i)
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				Namespace:         dep.Namespace,
				CreationTimestamp: metav1.NewTime(created),
				Labels:            labels,
			},
			Spec: corev1.PodSpec{
				NodeName:   "clustermind-demo-node",
				Containers: containers,
			},
		}
		if failing {
			pod.Status = corev1.PodStatus{
				Phase: corev1.PodPending,
				Conditions: []corev1.PodCondition{{
					Type:               corev1.PodReady,
					Status:             corev1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(now.Add(-10 * time.Minute)),
				}},
				ContainerStatuses: []corev1.ContainerStatus{{
					Name:         containers[0].Name,
					Image:        containers[0].Image,
					Ready:        false,
					RestartCount: 4,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "ImagePullBackOff",
							Message: `failed to pull image "nginx:does-not-exist": manifest unknown`,
						},
					},
				}},
			}
		} else {
			started := metav1.NewTime(created.Add(30 * time.Second))
			statuses := make([]corev1.ContainerStatus, 0, len(containers))
			for _, c := range containers {
				statuses = append(statuses, corev1.ContainerStatus{
					Name:         c.Name,
					Image:        c.Image,
					ImageID:      "docker.io/library/" + c.Name + "@sha256:1111111111111111111111111111111111111111111111111111111111111111",
					Ready:        true,
					RestartCount: 0,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{StartedAt: started},
					},
				})
			}
			pod.Status = corev1.PodStatus{
				Phase:     corev1.PodRunning,
				HostIP:    "10.0.0.5",
				PodIP:     fmt.Sprintf("10.244.0.%d", 10+int(i)),
				StartTime: &started,
				Conditions: []corev1.PodCondition{{
					Type:               corev1.PodReady,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(created.Add(30 * time.Second)),
				}},
				ContainerStatuses: statuses,
			}
		}
		pods = append(pods, pod)
	}
	return pods
}

// demoChangeEvents emits Normal ScalingReplicaSet events (the `changes`
// feed in the UI). ReplicaSet names carry the pod-template-hash suffix so
// resolveEventResource trims it back to the owning Deployment.
func demoChangeEvents(deployments []appsv1.Deployment, now time.Time) []corev1.Event {
	events := make([]corev1.Event, 0, 2)
	for _, dep := range deployments {
		if dep.Namespace != "demo" || (dep.Name != "frontend-deployment" && dep.Name != "api-deployment") {
			continue
		}
		rsName := fmt.Sprintf("%s-%s", dep.Name, demoPodTemplateHash)
		events = append(events, corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s.17a2f9c0d1e2b3", rsName),
				Namespace: dep.Namespace,
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:       "ReplicaSet",
				Namespace:  dep.Namespace,
				Name:       rsName,
				APIVersion: "apps/v1",
				UID:        types.UID("demo-uid-" + rsName),
			},
			Type:           corev1.EventTypeNormal,
			Reason:         "ScalingReplicaSet",
			Message:        fmt.Sprintf("Scaled up replica set %s to %d", rsName, replicaCount(dep)),
			Count:          1,
			FirstTimestamp: metav1.NewTime(now.Add(-45 * time.Minute)),
			LastTimestamp:  metav1.NewTime(now.Add(-45 * time.Minute)),
			EventTime:      metav1.NewMicroTime(now.Add(-45 * time.Minute)),
			Source:         corev1.EventSource{Component: "deployment-controller"},
		})
	}
	return events
}

// demoFailureEvents are the Warning events attached directly to the broken
// payment-deployment, matching a real ImagePullBackOff incident.
func demoFailureEvents(now time.Time) []corev1.Event {
	type fev struct {
		ago             time.Duration
		reason, message string
	}
	seq := []fev{
		{ago: 12 * time.Minute, reason: "Failed", message: `Failed to pull image "nginx:does-not-exist": [manifest unknown]`},
		{ago: 9 * time.Minute, reason: "Failed", message: "Error: ErrImagePull"},
		{ago: 3 * time.Minute, reason: "Failed", message: "Error: ImagePullBackOff"},
	}
	events := make([]corev1.Event, 0, len(seq))
	for i, s := range seq {
		t := now.Add(-s.ago)
		events = append(events, corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("payment-deployment.17a2f9c0d1e2b3%d", i),
				Namespace: "demo",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Deployment",
				Namespace:  "demo",
				Name:       "payment-deployment",
				APIVersion: "apps/v1",
				UID:        types.UID("demo-uid-payment-deployment"),
			},
			Type:           corev1.EventTypeWarning,
			Reason:         s.reason,
			Message:        s.message,
			Count:          1,
			FirstTimestamp: metav1.NewTime(t),
			LastTimestamp:  metav1.NewTime(t),
			EventTime:      metav1.NewMicroTime(t),
			Source:         corev1.EventSource{Component: "kubelet", Host: "clustermind-demo-node"},
		})
	}
	return events
}

func replicaCount(dep appsv1.Deployment) int32 {
	if dep.Spec.Replicas != nil {
		return *dep.Spec.Replicas
	}
	return 1
}

package probe

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsv1typed "k8s.io/client-go/kubernetes/typed/apps/v1"
	autoscalingv2typed "k8s.io/client-go/kubernetes/typed/autoscaling/v2"
	corev1typed "k8s.io/client-go/kubernetes/typed/core/v1"
	networkingv1typed "k8s.io/client-go/kubernetes/typed/networking/v1"
)

// mockClientset implements only the subsets of kubernetes.Interface called by ClusterCollector.
type mockClientset struct {
	kubernetes.Interface
	coreV1        *mockCoreV1
	appsV1        *mockAppsV1
	networkingV1  *mockNetworkingV1
	autoscalingV2 *mockAutoscalingV2
}

func (m *mockClientset) CoreV1() corev1typed.CoreV1Interface {
	if m.coreV1 != nil {
		return m.coreV1
	}
	return &mockCoreV1{}
}

func (m *mockClientset) AppsV1() appsv1typed.AppsV1Interface {
	if m.appsV1 != nil {
		return m.appsV1
	}
	return &mockAppsV1{}
}

func (m *mockClientset) NetworkingV1() networkingv1typed.NetworkingV1Interface {
	if m.networkingV1 != nil {
		return m.networkingV1
	}
	return &mockNetworkingV1{}
}

func (m *mockClientset) AutoscalingV2() autoscalingv2typed.AutoscalingV2Interface {
	if m.autoscalingV2 != nil {
		return m.autoscalingV2
	}
	return &mockAutoscalingV2{}
}

type mockCoreV1 struct {
	corev1typed.CoreV1Interface
	listNamespaces func(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error)
	listPods       func(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error)
	listServices   func(ctx context.Context, opts metav1.ListOptions) (*corev1.ServiceList, error)
	listConfigMaps func(ctx context.Context, opts metav1.ListOptions) (*corev1.ConfigMapList, error)
	listEvents     func(ctx context.Context, opts metav1.ListOptions) (*corev1.EventList, error)
	listSecrets    func(ctx context.Context, opts metav1.ListOptions) (*corev1.SecretList, error)
}

func (m *mockCoreV1) Namespaces() corev1typed.NamespaceInterface {
	return &mockNamespaceClient{listFunc: m.listNamespaces}
}
func (m *mockCoreV1) Pods(ns string) corev1typed.PodInterface {
	return &mockPodClient{listFunc: m.listPods}
}
func (m *mockCoreV1) Services(ns string) corev1typed.ServiceInterface {
	return &mockServiceClient{listFunc: m.listServices}
}
func (m *mockCoreV1) ConfigMaps(ns string) corev1typed.ConfigMapInterface {
	return &mockConfigMapClient{listFunc: m.listConfigMaps}
}
func (m *mockCoreV1) Events(ns string) corev1typed.EventInterface {
	return &mockEventClient{listFunc: m.listEvents}
}
func (m *mockCoreV1) Secrets(ns string) corev1typed.SecretInterface {
	return &mockSecretClient{listFunc: m.listSecrets}
}

type mockNamespaceClient struct {
	corev1typed.NamespaceInterface
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error)
}

func (c *mockNamespaceClient) List(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, opts)
	}
	return &corev1.NamespaceList{}, nil
}

type mockPodClient struct {
	corev1typed.PodInterface
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error)
}

func (c *mockPodClient) List(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, opts)
	}
	return &corev1.PodList{}, nil
}

type mockServiceClient struct {
	corev1typed.ServiceInterface
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*corev1.ServiceList, error)
}

func (c *mockServiceClient) List(ctx context.Context, opts metav1.ListOptions) (*corev1.ServiceList, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, opts)
	}
	return &corev1.ServiceList{}, nil
}

type mockConfigMapClient struct {
	corev1typed.ConfigMapInterface
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*corev1.ConfigMapList, error)
}

func (c *mockConfigMapClient) List(ctx context.Context, opts metav1.ListOptions) (*corev1.ConfigMapList, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, opts)
	}
	return &corev1.ConfigMapList{}, nil
}

type mockEventClient struct {
	corev1typed.EventInterface
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*corev1.EventList, error)
}

func (c *mockEventClient) List(ctx context.Context, opts metav1.ListOptions) (*corev1.EventList, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, opts)
	}
	return &corev1.EventList{}, nil
}

type mockSecretClient struct {
	corev1typed.SecretInterface
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*corev1.SecretList, error)
}

func (c *mockSecretClient) List(ctx context.Context, opts metav1.ListOptions) (*corev1.SecretList, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, opts)
	}
	return &corev1.SecretList{}, nil
}

type mockAppsV1 struct {
	appsv1typed.AppsV1Interface
	listDeployments  func(ctx context.Context, opts metav1.ListOptions) (*appsv1.DeploymentList, error)
	listStatefulSets func(ctx context.Context, opts metav1.ListOptions) (*appsv1.StatefulSetList, error)
	listDaemonSets   func(ctx context.Context, opts metav1.ListOptions) (*appsv1.DaemonSetList, error)
}

func (m *mockAppsV1) Deployments(ns string) appsv1typed.DeploymentInterface {
	return &mockDeploymentClient{listFunc: m.listDeployments}
}
func (m *mockAppsV1) StatefulSets(ns string) appsv1typed.StatefulSetInterface {
	return &mockStatefulSetClient{listFunc: m.listStatefulSets}
}
func (m *mockAppsV1) DaemonSets(ns string) appsv1typed.DaemonSetInterface {
	return &mockDaemonSetClient{listFunc: m.listDaemonSets}
}

type mockDeploymentClient struct {
	appsv1typed.DeploymentInterface
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*appsv1.DeploymentList, error)
}

func (c *mockDeploymentClient) List(ctx context.Context, opts metav1.ListOptions) (*appsv1.DeploymentList, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, opts)
	}
	return &appsv1.DeploymentList{}, nil
}

type mockStatefulSetClient struct {
	appsv1typed.StatefulSetInterface
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*appsv1.StatefulSetList, error)
}

func (c *mockStatefulSetClient) List(ctx context.Context, opts metav1.ListOptions) (*appsv1.StatefulSetList, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, opts)
	}
	return &appsv1.StatefulSetList{}, nil
}

type mockDaemonSetClient struct {
	appsv1typed.DaemonSetInterface
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*appsv1.DaemonSetList, error)
}

func (c *mockDaemonSetClient) List(ctx context.Context, opts metav1.ListOptions) (*appsv1.DaemonSetList, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, opts)
	}
	return &appsv1.DaemonSetList{}, nil
}

type mockNetworkingV1 struct {
	networkingv1typed.NetworkingV1Interface
	listIngresses       func(ctx context.Context, opts metav1.ListOptions) (*networkingv1.IngressList, error)
	listNetworkPolicies func(ctx context.Context, opts metav1.ListOptions) (*networkingv1.NetworkPolicyList, error)
}

func (m *mockNetworkingV1) Ingresses(ns string) networkingv1typed.IngressInterface {
	return &mockIngressClient{listFunc: m.listIngresses}
}
func (m *mockNetworkingV1) NetworkPolicies(ns string) networkingv1typed.NetworkPolicyInterface {
	return &mockNetworkPolicyClient{listFunc: m.listNetworkPolicies}
}

type mockIngressClient struct {
	networkingv1typed.IngressInterface
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*networkingv1.IngressList, error)
}

func (c *mockIngressClient) List(ctx context.Context, opts metav1.ListOptions) (*networkingv1.IngressList, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, opts)
	}
	return &networkingv1.IngressList{}, nil
}

type mockNetworkPolicyClient struct {
	networkingv1typed.NetworkPolicyInterface
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*networkingv1.NetworkPolicyList, error)
}

func (c *mockNetworkPolicyClient) List(ctx context.Context, opts metav1.ListOptions) (*networkingv1.NetworkPolicyList, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, opts)
	}
	return &networkingv1.NetworkPolicyList{}, nil
}

type mockAutoscalingV2 struct {
	autoscalingv2typed.AutoscalingV2Interface
	listHPAs func(ctx context.Context, opts metav1.ListOptions) (*autoscalingv2.HorizontalPodAutoscalerList, error)
}

func (m *mockAutoscalingV2) HorizontalPodAutoscalers(ns string) autoscalingv2typed.HorizontalPodAutoscalerInterface {
	return &mockHPAClient{listFunc: m.listHPAs}
}

type mockHPAClient struct {
	autoscalingv2typed.HorizontalPodAutoscalerInterface
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*autoscalingv2.HorizontalPodAutoscalerList, error)
}

func (c *mockHPAClient) List(ctx context.Context, opts metav1.ListOptions) (*autoscalingv2.HorizontalPodAutoscalerList, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, opts)
	}
	return &autoscalingv2.HorizontalPodAutoscalerList{}, nil
}

func TestCollector_OptionalResourceFailuresDoNotFailSnapshot(t *testing.T) {
	mockErr := errors.New("403 Forbidden: user cannot list resource")

	mock := &mockClientset{
		coreV1: &mockCoreV1{
			listNamespaces: func(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error) {
				return nil, mockErr
			},
			listPods: func(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
				return &corev1.PodList{
					Items: []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "web-pod-1", Namespace: "demo"}}},
				}, nil
			},
			listServices: func(ctx context.Context, opts metav1.ListOptions) (*corev1.ServiceList, error) {
				return &corev1.ServiceList{
					Items: []corev1.Service{{ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: "demo"}}},
				}, nil
			},
			listConfigMaps: func(ctx context.Context, opts metav1.ListOptions) (*corev1.ConfigMapList, error) {
				return nil, mockErr
			},
			listEvents: func(ctx context.Context, opts metav1.ListOptions) (*corev1.EventList, error) {
				return nil, mockErr
			},
		},
		appsV1: &mockAppsV1{
			listDeployments: func(ctx context.Context, opts metav1.ListOptions) (*appsv1.DeploymentList, error) {
				return &appsv1.DeploymentList{
					Items: []appsv1.Deployment{{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "demo"}}},
				}, nil
			},
			listStatefulSets: func(ctx context.Context, opts metav1.ListOptions) (*appsv1.StatefulSetList, error) {
				return nil, mockErr
			},
			listDaemonSets: func(ctx context.Context, opts metav1.ListOptions) (*appsv1.DaemonSetList, error) {
				return nil, mockErr
			},
		},
		networkingV1: &mockNetworkingV1{
			listIngresses: func(ctx context.Context, opts metav1.ListOptions) (*networkingv1.IngressList, error) {
				return nil, mockErr
			},
			listNetworkPolicies: func(ctx context.Context, opts metav1.ListOptions) (*networkingv1.NetworkPolicyList, error) {
				return nil, mockErr
			},
		},
		autoscalingV2: &mockAutoscalingV2{
			listHPAs: func(ctx context.Context, opts metav1.ListOptions) (*autoscalingv2.HorizontalPodAutoscalerList, error) {
				return nil, mockErr
			},
		},
	}

	collector := NewClusterCollector(mock, "demo")
	snap, err := collector.Collect(context.Background())

	require.NoError(t, err, "snapshot must not fail when optional resources fail")
	require.NotNil(t, snap)

	// Essential resources must be collected properly
	assert.Len(t, snap.Deployments, 1)
	assert.Equal(t, "web", snap.Deployments[0].Name)
	assert.Len(t, snap.Pods, 1)
	assert.Equal(t, "web-pod-1", snap.Pods[0].Name)
	assert.Len(t, snap.Services, 1)
	assert.Equal(t, "web-svc", snap.Services[0].Name)

	// Optional resources should simply be empty
	assert.Empty(t, snap.Namespaces)
	assert.Empty(t, snap.StatefulSets)
	assert.Empty(t, snap.DaemonSets)
	assert.Empty(t, snap.Ingresses)
	assert.Empty(t, snap.ConfigMaps)
	assert.Empty(t, snap.HPAs)
	assert.Empty(t, snap.NetworkPolicies)
	assert.Empty(t, snap.Events)

	// Secrets must never be collected
	assert.Empty(t, snap.Secrets)

	// CapturedAt must be set
	assert.WithinDuration(t, time.Now().UTC(), snap.CapturedAt, 5*time.Second)
}

func TestCollector_EssentialResourceFailuresReturnError(t *testing.T) {
	t.Run("Deployments failure is fatal", func(t *testing.T) {
		mock := &mockClientset{
			appsV1: &mockAppsV1{
				listDeployments: func(ctx context.Context, opts metav1.ListOptions) (*appsv1.DeploymentList, error) {
					return nil, errors.New("connection reset by peer")
				},
			},
		}

		collector := NewClusterCollector(mock, "demo")
		snap, err := collector.Collect(context.Background())
		require.Error(t, err)
		assert.Nil(t, snap)
		assert.Contains(t, err.Error(), "collect Deployment:")
	})

	t.Run("Pods failure is fatal", func(t *testing.T) {
		mock := &mockClientset{
			appsV1: &mockAppsV1{
				listDeployments: func(ctx context.Context, opts metav1.ListOptions) (*appsv1.DeploymentList, error) {
					return &appsv1.DeploymentList{}, nil
				},
			},
			coreV1: &mockCoreV1{
				listPods: func(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
					return nil, errors.New("pods unavailable")
				},
			},
		}

		collector := NewClusterCollector(mock, "demo")
		snap, err := collector.Collect(context.Background())
		require.Error(t, err)
		assert.Nil(t, snap)
		assert.Contains(t, err.Error(), "collect Pod:")
	})

	t.Run("Services failure is fatal", func(t *testing.T) {
		mock := &mockClientset{
			appsV1: &mockAppsV1{
				listDeployments: func(ctx context.Context, opts metav1.ListOptions) (*appsv1.DeploymentList, error) {
					return &appsv1.DeploymentList{}, nil
				},
			},
			coreV1: &mockCoreV1{
				listPods: func(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
					return &corev1.PodList{}, nil
				},
				listServices: func(ctx context.Context, opts metav1.ListOptions) (*corev1.ServiceList, error) {
					return nil, errors.New("services unavailable")
				},
			},
		}

		collector := NewClusterCollector(mock, "demo")
		snap, err := collector.Collect(context.Background())
		require.Error(t, err)
		assert.Nil(t, snap)
		assert.Contains(t, err.Error(), "collect Service:")
	})
}

func TestCollector_SecretsAreNeverListed(t *testing.T) {
	var secretListCount int32
	mock := &mockClientset{
		coreV1: &mockCoreV1{
			listPods: func(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
				return &corev1.PodList{}, nil
			},
			listServices: func(ctx context.Context, opts metav1.ListOptions) (*corev1.ServiceList, error) {
				return &corev1.ServiceList{}, nil
			},
			listSecrets: func(ctx context.Context, opts metav1.ListOptions) (*corev1.SecretList, error) {
				atomic.AddInt32(&secretListCount, 1)
				return nil, errors.New("secrets access forbidden")
			},
		},
		appsV1: &mockAppsV1{
			listDeployments: func(ctx context.Context, opts metav1.ListOptions) (*appsv1.DeploymentList, error) {
				return &appsv1.DeploymentList{}, nil
			},
		},
	}

	collector := NewClusterCollector(mock, "demo")
	snap, err := collector.Collect(context.Background())

	require.NoError(t, err)
	require.NotNil(t, snap)
	assert.Equal(t, int32(0), atomic.LoadInt32(&secretListCount), "collector must not attempt to list secrets")
	assert.Empty(t, snap.Secrets, "snapshot secrets must remain empty")
}

/*
Copyright 2022 The Kruise Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hwcloud

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gamekruiseiov1alpha1 "github.com/openkruise/kruise-game/apis/v1alpha1"
	"github.com/openkruise/kruise-game/cloudprovider/errors"
)

func TestAllocateDeAllocate(t *testing.T) {
	test := struct {
		lbIds  []string
		elb    *ElbPlugin
		num    int
		podKey string
	}{
		lbIds: []string{"cce-lb-xxxx"},
		elb: &ElbPlugin{
			maxPort:     int32(712),
			minPort:     int32(512),
			cache:       make(map[string]portAllocated),
			podAllocate: make(map[string]string),
			mutex:       sync.RWMutex{},
		},
		podKey: "xxx/xxx",
		num:    3,
	}

	lbId, ports := test.elb.allocate(test.lbIds, test.num, test.podKey)
	if _, exist := test.elb.podAllocate[test.podKey]; !exist {
		t.Errorf("podAllocate[%s] is empty after allocated", test.podKey)
	}
	for _, port := range ports {
		if port > test.elb.maxPort || port < test.elb.minPort {
			t.Errorf("allocate port %d, unexpected", port)
		}
		if test.elb.cache[lbId][port] == false {
			t.Errorf("Allocate port %d failed", port)
		}
	}
	test.elb.deAllocate(test.podKey)
	for _, port := range ports {
		if test.elb.cache[lbId][port] == true {
			t.Errorf("deAllocate port %d failed", port)
		}
	}
	if _, exist := test.elb.podAllocate[test.podKey]; exist {
		t.Errorf("podAllocate[%s] is not empty after deallocated", test.podKey)
	}
}

func TestParseLbConfig(t *testing.T) {
	tests := []struct {
		conf      []gamekruiseiov1alpha1.NetworkConfParams
		slbConfig *elbConfig
	}{
		{
			conf: []gamekruiseiov1alpha1.NetworkConfParams{
				{
					Name:  PortProtocolsConfigName,
					Value: "80",
				},
				{
					Name:  "kubernetes.io/elb.class",
					Value: "performance",
				},
				{
					Name:  "kubernetes.io/elb.id",
					Value: "c1d8f4c6-7aef-4596-8c7c-xxxxxxxxxxxx",
				},
				{
					Name:  "kubernetes.io/elb.enterpriseID",
					Value: "ff97261-4dbd-4593-8236-xxxxxxxxxxxx",
				},
				{
					Name:  "kubernetes.io/elb.lb-algorithm",
					Value: "ROUND_ROBIN",
				},
				{
					Name:  "kubernetes.io/elb.protocol-port",
					Value: "https:443,http:80",
				},
				{
					Name:  "kubernetes.io/elb.x-forwarded-port",
					Value: "true",
				},
			},
			slbConfig: &elbConfig{
				elbIds:                    []string{"c1d8f4c6-7aef-4596-8c7c-xxxxxxxxxxxx"},
				targetPorts:               []int{80},
				protocols:                 []corev1.Protocol{corev1.ProtocolTCP},
				externalTrafficPolicyType: corev1.ServiceExternalTrafficPolicyTypeCluster,
				isFixed:                   false,
				hwOptions: map[string]string{
					"kubernetes.io/elb.class":            "performance",
					"kubernetes.io/elb.id":               "c1d8f4c6-7aef-4596-8c7c-xxxxxxxxxxxx",
					"kubernetes.io/elb.enterpriseID":     "ff97261-4dbd-4593-8236-xxxxxxxxxxxx",
					"kubernetes.io/elb.lb-algorithm":     "ROUND_ROBIN",
					"kubernetes.io/elb.protocol-port":    "https:443,http:80",
					"kubernetes.io/elb.x-forwarded-port": "true",
				},
			},
		},
		{
			conf: []gamekruiseiov1alpha1.NetworkConfParams{
				{
					Name:  PortProtocolsConfigName,
					Value: "81/UDP,82,83/TCP",
				},
				{
					Name:  "kubernetes.io/elb.class",
					Value: "union",
				},
				{
					Name:  "kubernetes.io/elb.id",
					Value: "c1d8f4c6-7aef-4596-8c7c-yyyyyyyyyyyy",
				},
				{
					Name:  "kubernetes.io/elb.enterpriseID",
					Value: "ff97261-4dbd-4593-8236-yyyyyyyyyyyy",
				},
				{
					Name:  "kubernetes.io/elb.cert-id",
					Value: "17e3b4f4bc40471c86741dc3aa211379",
				},
				{
					Name:  "kubernetes.io/elb.tls-certificate-ids",
					Value: "5196aa70b0f143189e4cb54991ba2286,8125d71fcc124aabbe007610cba42d60",
				},
				{
					Name:  "kubernetes.io/elb.multicluster",
					Value: "true",
				},
				{
					Name:  "kubernetes.io/elb.keepalive_timeout",
					Value: "400s",
				},
				{
					Name:  "kubernetes.io/elb.client_timeout",
					Value: "50s",
				},
				{
					Name:  "kubernetes.io/elb.member_timeout",
					Value: "50s",
				},
				{
					Name:  ExternalTrafficPolicyTypeConfigName,
					Value: "Local",
				},
			},
			slbConfig: &elbConfig{
				elbIds:                    []string{"c1d8f4c6-7aef-4596-8c7c-yyyyyyyyyyyy"},
				targetPorts:               []int{81, 82, 83},
				protocols:                 []corev1.Protocol{corev1.ProtocolUDP, corev1.ProtocolTCP, corev1.ProtocolTCP},
				externalTrafficPolicyType: corev1.ServiceExternalTrafficPolicyTypeLocal,
				isFixed:                   false,
				hwOptions: map[string]string{
					"kubernetes.io/elb.class":               "union",
					"kubernetes.io/elb.id":                  "c1d8f4c6-7aef-4596-8c7c-yyyyyyyyyyyy",
					"kubernetes.io/elb.enterpriseID":        "ff97261-4dbd-4593-8236-yyyyyyyyyyyy",
					"kubernetes.io/elb.cert-id":             "17e3b4f4bc40471c86741dc3aa211379",
					"kubernetes.io/elb.tls-certificate-ids": "5196aa70b0f143189e4cb54991ba2286,8125d71fcc124aabbe007610cba42d60",
					"kubernetes.io/elb.multicluster":        "true",
					"kubernetes.io/elb.keepalive_timeout":   "400s",
					"kubernetes.io/elb.client_timeout":      "50s",
					"kubernetes.io/elb.member_timeout":      "50s",
				},
			},
		},
	}

	for i, test := range tests {
		sc, err := parseLbConfig(test.conf)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(test.slbConfig, sc) {
			t.Errorf("case %d: lbId expect: %v, actual: %v", i, test.slbConfig, sc)
		}
	}
}

func TestInitLbCache(t *testing.T) {
	test := struct {
		svcList     []corev1.Service
		minPort     int32
		maxPort     int32
		blockPorts  []int32
		cache       map[string]portAllocated
		podAllocate map[string]string
	}{
		minPort:    512,
		maxPort:    712,
		blockPorts: []int32{593},
		cache: map[string]portAllocated{
			"xxx-A": map[int32]bool{
				666: true,
				593: true,
			},
			"xxx-B": map[int32]bool{
				555: true,
				593: true,
			},
		},
		podAllocate: map[string]string{
			"ns-0/name-0": "xxx-A:666",
			"ns-1/name-1": "xxx-B:555",
		},
		svcList: []corev1.Service{
			{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ElbIdAnnotationKey: "xxx-A",
					},
					Namespace: "ns-0",
					Name:      "name-0",
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
					Selector: map[string]string{
						SvcSelectorKey: "pod-A",
					},
					Ports: []corev1.ServicePort{
						{
							TargetPort: intstr.FromInt(80),
							Port:       666,
							Protocol:   corev1.ProtocolTCP,
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ElbIdAnnotationKey: "xxx-B",
					},
					Namespace: "ns-1",
					Name:      "name-1",
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
					Selector: map[string]string{
						SvcSelectorKey: "pod-B",
					},
					Ports: []corev1.ServicePort{
						{
							TargetPort: intstr.FromInt(8080),
							Port:       555,
							Protocol:   corev1.ProtocolTCP,
						},
					},
				},
			},
		},
	}

	actualCache, actualPodAllocate := initLbCache(test.svcList, test.minPort, test.maxPort, test.blockPorts)
	for lb, pa := range test.cache {
		for port, isAllocated := range pa {
			if actualCache[lb][port] != isAllocated {
				t.Errorf("lb %s port %d isAllocated, expect: %t, actual: %t", lb, port, isAllocated, actualCache[lb][port])
			}
		}
	}
	if !reflect.DeepEqual(actualPodAllocate, test.podAllocate) {
		t.Errorf("podAllocate expect %v, but actully got %v", test.podAllocate, actualPodAllocate)
	}
}

func TestElbPlugin_OnPodUpdated(t *testing.T) {
	type fields struct {
		maxPort     int32
		minPort     int32
		blockPorts  []int32
		cache       map[string]portAllocated
		podAllocate map[string]string
		mutex       sync.RWMutex
	}
	type args struct {
		c   client.Client
		pod func() *corev1.Pod
		ctx context.Context
	}
	networkNotReadyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-0",
			Namespace: "default",
			UID:       "53cb0992-c720-4ae4-9af9-3cc7e2bf3660",
			Annotations: map[string]string{
				"game.kruise.io/network-type": ElbNetwork,
				"game.kruise.io/network-conf": `[{"name":"PortProtocols","value":"80/TCP"},{"name":"kubernetes.io/elb.class","value":"performance"},{"name":"kubernetes.io/elb.id","value":"8f4cf216-a659-40dc-8c77-6068b036ba56"},{"name":"kubernetes.io/elb.connection-drain-enable","value":"true"},{"name":"kubernetes.io/elb.connection-drain-timeout","value":"300"}]`,
			},
		},
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		setup  func(*MockClient, *MockNetworkManager)
		want   func() *corev1.Pod
		want1  errors.PluginError
	}{
		{
			name: "network is not ready",
			fields: fields{
				maxPort:     500,
				minPort:     502,
				blockPorts:  []int32{501},
				cache:       map[string]portAllocated{"8f4cf216-a659-40dc-8c77-6068b036ba56": map[int32]bool{500: true, 501: true, 502: false}},
				podAllocate: map[string]string{"default/test-pod-0": "8f4cf216-a659-40dc-8c77-6068b036ba56:501"},
				mutex:       sync.RWMutex{},
			},
			args: args{
				c: new(MockClient),
				pod: func() *corev1.Pod {
					return networkNotReadyPod
				},
				ctx: context.Background(),
			},
			//want: networkNotReadyPodWant,
			want: func() *corev1.Pod {
				res := networkNotReadyPod.DeepCopy()
				res.Annotations["game.kruise.io/network-status"] = `{"currentNetworkState":"NotReady","createTime":null,"lastTransitionTime":null}`
				return res
			},
			want1: nil,
		},
		{
			name: "network is ready",
			fields: fields{
				maxPort:     500,
				minPort:     502,
				blockPorts:  []int32{501},
				cache:       map[string]portAllocated{"8f4cf216-a659-40dc-8c77-6068b036ba56": map[int32]bool{500: true, 501: true, 502: false}},
				podAllocate: map[string]string{"default/test-pod-0": "8f4cf216-a659-40dc-8c77-6068b036ba56:500,501"},
				mutex:       sync.RWMutex{},
			},
			args: args{
				c: nil,
				pod: func() *corev1.Pod {
					res := networkNotReadyPod.DeepCopy()
					res.Annotations["game.kruise.io/network-status"] = `{"internalAddresses":[{"ip":"192.168.1.38","ports":[{"name":"80","protocol":"TCP","port":80}]}],"externalAddresses":[{"ip":"192.168.0.147","ports":[{"name":"80","protocol":"TCP","port":500}]}],"currentNetworkState":"Ready","createTime":null,"lastTransitionTime":null}`
					return res
				},
				ctx: context.Background(),
			},
			setup: func(clientMock *MockClient, nmMock *MockNetworkManager) {
				nmMock.On("GetNetworkType").Return(ElbNetwork)
				clientMock.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					// 可选：填充一个 Service 实例作为输出
					service := args[2].(*corev1.Service)
					*service = corev1.Service{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Service",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "gs-elb-performance-2-0",
							Namespace: "kruise-game-system",
							Annotations: map[string]string{
								"game.kruise.io/network-config-hash":         "1234914076",
								"kubernetes.io/elb.class":                    "performance",
								"kubernetes.io/elb.connection-drain-enable":  "true",
								"kubernetes.io/elb.connection-drain-timeout": "300",
								"kubernetes.io/elb.id":                       "8f4cf216-a659-40dc-8c77-6068b036ba56",
								"kubernetes.io/elb.mark":                     "0",
							},
							CreationTimestamp: metav1.Time{},
							Finalizers:        []string{"service.kubernetes.io/load-balancer-cleanup"},
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: "v1",
									Kind:       "Pod",
									Name:       "gs-elb-performance-2-0",
									UID:        "53cb0992-c720-4ae4-9af9-3cc7e2bf3660",
								},
							},
							ResourceVersion: "9867633",
							UID:             "eccc0de3-ea09-4554-8710-4319eb551237",
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									Name:       "80-tcp",
									Protocol:   corev1.ProtocolTCP,
									Port:       500,
									TargetPort: intstr.FromInt(80),
									NodePort:   31749,
								},
							},
							Selector: map[string]string{
								"statefulset.kubernetes.io/pod-name": "gs-elb-performance-2-0",
							},
							ClusterIP:             "10.247.83.164",
							ClusterIPs:            []string{"10.247.83.164"},
							Type:                  corev1.ServiceTypeLoadBalancer,
							ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
							LoadBalancerIP:        "192.168.0.147",
							IPFamilies:            []corev1.IPFamily{corev1.IPv4Protocol},
							SessionAffinity:       corev1.ServiceAffinityNone,
						},
						Status: corev1.ServiceStatus{
							LoadBalancer: corev1.LoadBalancerStatus{
								Ingress: []corev1.LoadBalancerIngress{
									{IP: "192.168.0.147"},
									{IP: "189.1.225.136"},
								},
							},
						},
					}
				}).Return(nil)
			},
			want: func() *corev1.Pod {
				res := networkNotReadyPod.DeepCopy()
				res.Annotations["game.kruise.io/network-status"] = `{"internalAddresses":[{"ip":"","ports":[{"name":"80","protocol":"TCP","port":80}]}],"externalAddresses":[{"ip":"192.168.0.147","ports":[{"name":"80","protocol":"TCP","port":500}]}],"currentNetworkState":"Ready","createTime":null,"lastTransitionTime":null}`
				res.Annotations["game.kruise.io/network-conf"] = `[{"name":"PortProtocols","value":"80/TCP"},{"name":"kubernetes.io/elb.class","value":"performance"},{"name":"kubernetes.io/elb.id","value":"8f4cf216-a659-40dc-8c77-6068b036ba56"},{"name":"kubernetes.io/elb.connection-drain-enable","value":"true"},{"name":"kubernetes.io/elb.connection-drain-timeout","value":"300"}]`
				return res
			},
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientMock := new(MockClient)
			nmMock := new(MockNetworkManager)
			if tt.setup != nil {
				tt.setup(clientMock, nmMock)
			}
			if tt.want == nil {
				t.Fatal("want is nil, set the function")
			}
			if tt.args.pod == nil {
				t.Fatal("pod is nil, set the function")
			}
			s := &ElbPlugin{
				maxPort:     tt.fields.maxPort,
				minPort:     tt.fields.minPort,
				blockPorts:  tt.fields.blockPorts,
				cache:       tt.fields.cache,
				podAllocate: tt.fields.podAllocate,
				mutex:       tt.fields.mutex,
			}
			tt.args.c = clientMock
			got, got1 := s.OnPodUpdated(tt.args.c, tt.args.pod(), tt.args.ctx)
			assert.Equalf(t, tt.want().Annotations["game.kruise.io/network-status"], got.Annotations["game.kruise.io/network-status"], "OnPodUpdated(%v, %v, %v)", tt.args.c, tt.args.pod, tt.args.ctx)
			assert.Equalf(t, tt.want().Annotations["game.kruise.io/network-type"], got.Annotations["game.kruise.io/network-type"], "OnPodUpdated(%v, %v, %v)", tt.args.c, tt.args.pod, tt.args.ctx)
			assert.Equalf(t, tt.want().Annotations["game.kruise.io/network-conf"], got.Annotations["game.kruise.io/network-conf"], "OnPodUpdated(%v, %v, %v)", tt.args.c, tt.args.pod, tt.args.ctx)
			assert.Equalf(t, tt.want1, got1, "OnPodUpdated(%v, %v, %v)", tt.args.c, tt.args.pod, tt.args.ctx)
		})
	}
}

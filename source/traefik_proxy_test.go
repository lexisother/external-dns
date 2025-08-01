/*
Copyright 2022 The Kubernetes Authors.

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

package source

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	fakeDynamic "k8s.io/client-go/dynamic/fake"
	fakeKube "k8s.io/client-go/kubernetes/fake"

	"sigs.k8s.io/external-dns/endpoint"
)

// This is a compile-time validation that traefikSource is a Source.
var _ Source = &traefikSource{}

const defaultTraefikNamespace = "traefik"

func TestTraefikProxyIngressRouteEndpoints(t *testing.T) {
	t.Parallel()

	for _, ti := range []struct {
		title                    string
		ingressRoute             IngressRoute
		ignoreHostnameAnnotation bool
		expected                 []*endpoint.Endpoint
	}{
		{
			title: "IngressRoute with hostname annotation",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "a.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-annotation",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute with host rule",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-host-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteSpec{
					Routes: []traefikRoute{
						{
							Match: "Host(`b.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "b.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-host-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute with hostheader rule",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-hostheader-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteSpec{
					Routes: []traefikRoute{
						{
							Match: "HostHeader(`c.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "c.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-hostheader-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute with multiple host rules",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-multi-host-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteSpec{
					Routes: []traefikRoute{
						{
							Match: "Host(`d.example.com`) || Host(`e.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "d.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "e.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute with multiple host rules and annotation",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-multi-host-annotations-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "f.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
				Spec: traefikIngressRouteSpec{
					Routes: []traefikRoute{
						{
							Match: "Host(`g.example.com`, `h.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "f.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "g.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "h.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute ignoring annotation",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-multi-host-annotations-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "f.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
				Spec: traefikIngressRouteSpec{
					Routes: []traefikRoute{
						{
							Match: "Host(`g.example.com`, `h.example.com`)",
						},
					},
				},
			},
			ignoreHostnameAnnotation: true,
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "g.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "h.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute omit wildcard",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-omit-wildcard-host",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteSpec{
					Routes: []traefikRoute{
						{
							Match: "Host(`*`)",
						},
					},
				},
			},
			expected: nil,
		},
	} {
		t.Run(ti.title, func(t *testing.T) {
			t.Parallel()

			fakeKubernetesClient := fakeKube.NewSimpleClientset()
			scheme := runtime.NewScheme()
			scheme.AddKnownTypes(ingressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(ingressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(ingressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			scheme.AddKnownTypes(oldIngressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(oldIngressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(oldIngressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			fakeDynamicClient := fakeDynamic.NewSimpleDynamicClient(scheme)

			ir := unstructured.Unstructured{}

			ingressRouteAsJSON, err := json.Marshal(ti.ingressRoute)
			assert.NoError(t, err)

			assert.NoError(t, ir.UnmarshalJSON(ingressRouteAsJSON))

			// Create proxy resources
			_, err = fakeDynamicClient.Resource(ingressRouteGVR).Namespace(defaultTraefikNamespace).Create(context.Background(), &ir, metav1.CreateOptions{})
			assert.NoError(t, err)

			source, err := NewTraefikSource(context.TODO(), fakeDynamicClient, fakeKubernetesClient, defaultTraefikNamespace, "kubernetes.io/ingress.class=traefik", ti.ignoreHostnameAnnotation, false, false)
			assert.NoError(t, err)
			assert.NotNil(t, source)

			count := &unstructured.UnstructuredList{}
			for len(count.Items) < 1 {
				count, _ = fakeDynamicClient.Resource(ingressRouteGVR).Namespace(defaultTraefikNamespace).List(context.Background(), metav1.ListOptions{})
			}

			endpoints, err := source.Endpoints(context.Background())
			assert.NoError(t, err)
			assert.Len(t, endpoints, len(ti.expected))
			assert.Equal(t, ti.expected, endpoints)
		})
	}
}

func TestTraefikProxyIngressRouteTCPEndpoints(t *testing.T) {
	t.Parallel()

	for _, ti := range []struct {
		title                    string
		ingressRouteTCP          IngressRouteTCP
		ignoreHostnameAnnotation bool
		expected                 []*endpoint.Endpoint
	}{
		{
			title: "IngressRouteTCP with hostname annotation",
			ingressRouteTCP: IngressRouteTCP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteTCPGVR.GroupVersion().String(),
					Kind:       "IngressRouteTCP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroutetcp-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "a.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-annotation",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP with host sni rule",
			ingressRouteTCP: IngressRouteTCP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteTCPGVR.GroupVersion().String(),
					Kind:       "IngressRouteTCP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroutetcp-hostsni-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteTCPSpec{
					Routes: []traefikRouteTCP{
						{
							Match: "HostSNI(`b.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "b.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-hostsni-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP with multiple host sni rules",
			ingressRouteTCP: IngressRouteTCP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteTCPGVR.GroupVersion().String(),
					Kind:       "IngressRouteTCP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroutetcp-multi-host-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteTCPSpec{
					Routes: []traefikRouteTCP{
						{
							Match: "HostSNI(`d.example.com`) || HostSNI(`e.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "d.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "e.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP with multiple host sni rules and annotation",
			ingressRouteTCP: IngressRouteTCP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteTCPGVR.GroupVersion().String(),
					Kind:       "IngressRouteTCP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroutetcp-multi-host-annotations-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "f.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
				Spec: traefikIngressRouteTCPSpec{
					Routes: []traefikRouteTCP{
						{
							Match: "HostSNI(`g.example.com`, `h.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "f.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "g.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "h.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP ignoring annotation",
			ingressRouteTCP: IngressRouteTCP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteTCPGVR.GroupVersion().String(),
					Kind:       "IngressRouteTCP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroutetcp-multi-host-annotations-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "f.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
				Spec: traefikIngressRouteTCPSpec{
					Routes: []traefikRouteTCP{
						{
							Match: "HostSNI(`g.example.com`, `h.example.com`)",
						},
					},
				},
			},
			ignoreHostnameAnnotation: true,
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "g.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "h.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP omit wildcard host sni",
			ingressRouteTCP: IngressRouteTCP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteTCPGVR.GroupVersion().String(),
					Kind:       "IngressRouteTCP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroutetcp-omit-wildcard-host",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteTCPSpec{
					Routes: []traefikRouteTCP{
						{
							Match: "HostSNI(`*`)",
						},
					},
				},
			},
			expected: nil,
		},
	} {
		t.Run(ti.title, func(t *testing.T) {
			t.Parallel()

			fakeKubernetesClient := fakeKube.NewSimpleClientset()
			scheme := runtime.NewScheme()
			scheme.AddKnownTypes(ingressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(ingressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(ingressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			scheme.AddKnownTypes(oldIngressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(oldIngressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(oldIngressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			fakeDynamicClient := fakeDynamic.NewSimpleDynamicClient(scheme)

			ir := unstructured.Unstructured{}

			ingressRouteAsJSON, err := json.Marshal(ti.ingressRouteTCP)
			require.NoError(t, err)

			require.NoError(t, ir.UnmarshalJSON(ingressRouteAsJSON))

			// Create proxy resources
			_, err = fakeDynamicClient.Resource(ingressRouteTCPGVR).Namespace(defaultTraefikNamespace).Create(context.Background(), &ir, metav1.CreateOptions{})
			require.NoError(t, err)

			source, err := NewTraefikSource(context.TODO(), fakeDynamicClient, fakeKubernetesClient, defaultTraefikNamespace, "kubernetes.io/ingress.class=traefik", ti.ignoreHostnameAnnotation, false, false)
			require.NoError(t, err)
			assert.NotNil(t, source)

			count := &unstructured.UnstructuredList{}
			for len(count.Items) < 1 {
				count, _ = fakeDynamicClient.Resource(ingressRouteTCPGVR).Namespace(defaultTraefikNamespace).List(context.Background(), metav1.ListOptions{})
			}

			endpoints, err := source.Endpoints(context.Background())
			require.NoError(t, err)
			assert.Len(t, endpoints, len(ti.expected))
			assert.Equal(t, ti.expected, endpoints)
		})
	}
}

func TestTraefikProxyIngressRouteUDPEndpoints(t *testing.T) {
	t.Parallel()

	for _, ti := range []struct {
		title                    string
		ingressRouteUDP          IngressRouteUDP
		ignoreHostnameAnnotation bool
		expected                 []*endpoint.Endpoint
	}{
		{
			title: "IngressRouteTCP with hostname annotation",
			ingressRouteUDP: IngressRouteUDP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteUDPGVR.GroupVersion().String(),
					Kind:       "IngressRouteUDP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressrouteudp-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "a.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressrouteudp/traefik/ingressrouteudp-annotation",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP with multiple hostname annotation",
			ingressRouteUDP: IngressRouteUDP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteUDPGVR.GroupVersion().String(),
					Kind:       "IngressRouteUDP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressrouteudp-multi-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com, b.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "a.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressrouteudp/traefik/ingressrouteudp-multi-annotation",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "b.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressrouteudp/traefik/ingressrouteudp-multi-annotation",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP ignoring hostname annotation",
			ingressRouteUDP: IngressRouteUDP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteUDPGVR.GroupVersion().String(),
					Kind:       "IngressRouteUDP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressrouteudp-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			ignoreHostnameAnnotation: true,
			expected:                 nil,
		},
	} {
		t.Run(ti.title, func(t *testing.T) {
			t.Parallel()

			fakeKubernetesClient := fakeKube.NewSimpleClientset()
			scheme := runtime.NewScheme()
			scheme.AddKnownTypes(ingressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(ingressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(ingressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			scheme.AddKnownTypes(oldIngressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(oldIngressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(oldIngressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			fakeDynamicClient := fakeDynamic.NewSimpleDynamicClient(scheme)

			ir := unstructured.Unstructured{}

			ingressRouteAsJSON, err := json.Marshal(ti.ingressRouteUDP)
			assert.NoError(t, err)

			assert.NoError(t, ir.UnmarshalJSON(ingressRouteAsJSON))

			// Create proxy resources
			_, err = fakeDynamicClient.Resource(ingressRouteUDPGVR).Namespace(defaultTraefikNamespace).Create(context.Background(), &ir, metav1.CreateOptions{})
			assert.NoError(t, err)

			source, err := NewTraefikSource(context.TODO(), fakeDynamicClient, fakeKubernetesClient, defaultTraefikNamespace, "kubernetes.io/ingress.class=traefik", ti.ignoreHostnameAnnotation, false, false)
			assert.NoError(t, err)
			assert.NotNil(t, source)

			count := &unstructured.UnstructuredList{}
			for len(count.Items) < 1 {
				count, _ = fakeDynamicClient.Resource(ingressRouteUDPGVR).Namespace(defaultTraefikNamespace).List(context.Background(), metav1.ListOptions{})
			}

			endpoints, err := source.Endpoints(context.Background())
			assert.NoError(t, err)
			assert.Len(t, endpoints, len(ti.expected))
			assert.Equal(t, ti.expected, endpoints)
		})
	}
}

func TestTraefikProxyOldIngressRouteEndpoints(t *testing.T) {
	t.Parallel()

	for _, ti := range []struct {
		title                    string
		ingressRoute             IngressRoute
		ignoreHostnameAnnotation bool
		expected                 []*endpoint.Endpoint
	}{
		{
			title: "IngressRoute with hostname annotation",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "a.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-annotation",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute with host rule",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-host-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteSpec{
					Routes: []traefikRoute{
						{
							Match: "Host(`b.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "b.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-host-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute with hostheader rule",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-hostheader-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteSpec{
					Routes: []traefikRoute{
						{
							Match: "HostHeader(`c.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "c.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-hostheader-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute with multiple host rules",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-multi-host-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteSpec{
					Routes: []traefikRoute{
						{
							Match: "Host(`d.example.com`) || Host(`e.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "d.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "e.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute with multiple host rules and annotation",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-multi-host-annotations-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "f.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
				Spec: traefikIngressRouteSpec{
					Routes: []traefikRoute{
						{
							Match: "Host(`g.example.com`, `h.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "f.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "g.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "h.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute ignoring annotation",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-multi-host-annotations-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "f.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
				Spec: traefikIngressRouteSpec{
					Routes: []traefikRoute{
						{
							Match: "Host(`g.example.com`, `h.example.com`)",
						},
					},
				},
			},
			ignoreHostnameAnnotation: true,
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "g.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "h.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute omit wildcard",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-omit-wildcard-host",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteSpec{
					Routes: []traefikRoute{
						{
							Match: "Host(`*`)",
						},
					},
				},
			},
			expected: nil,
		},
	} {
		t.Run(ti.title, func(t *testing.T) {
			t.Parallel()

			fakeKubernetesClient := fakeKube.NewSimpleClientset()
			scheme := runtime.NewScheme()
			scheme.AddKnownTypes(ingressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(ingressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(ingressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			scheme.AddKnownTypes(oldIngressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(oldIngressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(oldIngressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			fakeDynamicClient := fakeDynamic.NewSimpleDynamicClient(scheme)

			ir := unstructured.Unstructured{}

			ingressRouteAsJSON, err := json.Marshal(ti.ingressRoute)
			assert.NoError(t, err)

			assert.NoError(t, ir.UnmarshalJSON(ingressRouteAsJSON))

			// Create proxy resources
			_, err = fakeDynamicClient.Resource(oldIngressRouteGVR).Namespace(defaultTraefikNamespace).Create(context.Background(), &ir, metav1.CreateOptions{})
			assert.NoError(t, err)

			source, err := NewTraefikSource(context.TODO(), fakeDynamicClient, fakeKubernetesClient, defaultTraefikNamespace, "kubernetes.io/ingress.class=traefik", ti.ignoreHostnameAnnotation, true, false)
			assert.NoError(t, err)
			assert.NotNil(t, source)

			count := &unstructured.UnstructuredList{}
			for len(count.Items) < 1 {
				count, _ = fakeDynamicClient.Resource(oldIngressRouteGVR).Namespace(defaultTraefikNamespace).List(context.Background(), metav1.ListOptions{})
			}

			endpoints, err := source.Endpoints(context.Background())
			assert.NoError(t, err)
			assert.Len(t, endpoints, len(ti.expected))
			assert.Equal(t, ti.expected, endpoints)
		})
	}
}

func TestTraefikProxyOldIngressRouteTCPEndpoints(t *testing.T) {
	t.Parallel()

	for _, ti := range []struct {
		title                    string
		ingressRouteTCP          IngressRouteTCP
		ignoreHostnameAnnotation bool
		expected                 []*endpoint.Endpoint
	}{
		{
			title: "IngressRouteTCP with hostname annotation",
			ingressRouteTCP: IngressRouteTCP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteTCPGVR.GroupVersion().String(),
					Kind:       "IngressRouteTCP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroutetcp-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "a.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-annotation",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP with host sni rule",
			ingressRouteTCP: IngressRouteTCP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteTCPGVR.GroupVersion().String(),
					Kind:       "IngressRouteTCP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroutetcp-hostsni-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteTCPSpec{
					Routes: []traefikRouteTCP{
						{
							Match: "HostSNI(`b.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "b.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-hostsni-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP with multiple host sni rules",
			ingressRouteTCP: IngressRouteTCP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteTCPGVR.GroupVersion().String(),
					Kind:       "IngressRouteTCP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroutetcp-multi-host-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteTCPSpec{
					Routes: []traefikRouteTCP{
						{
							Match: "HostSNI(`d.example.com`) || HostSNI(`e.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "d.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "e.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP with multiple host sni rules and annotation",
			ingressRouteTCP: IngressRouteTCP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteTCPGVR.GroupVersion().String(),
					Kind:       "IngressRouteTCP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroutetcp-multi-host-annotations-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "f.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
				Spec: traefikIngressRouteTCPSpec{
					Routes: []traefikRouteTCP{
						{
							Match: "HostSNI(`g.example.com`, `h.example.com`)",
						},
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "f.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "g.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "h.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP ignoring annotation",
			ingressRouteTCP: IngressRouteTCP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteTCPGVR.GroupVersion().String(),
					Kind:       "IngressRouteTCP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroutetcp-multi-host-annotations-match",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "f.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
				Spec: traefikIngressRouteTCPSpec{
					Routes: []traefikRouteTCP{
						{
							Match: "HostSNI(`g.example.com`, `h.example.com`)",
						},
					},
				},
			},
			ignoreHostnameAnnotation: true,
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "g.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "h.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroutetcp/traefik/ingressroutetcp-multi-host-annotations-match",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP omit wildcard host sni",
			ingressRouteTCP: IngressRouteTCP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteTCPGVR.GroupVersion().String(),
					Kind:       "IngressRouteTCP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroutetcp-omit-wildcard-host",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/target": "target.domain.tld",
						"kubernetes.io/ingress.class":             "traefik",
					},
				},
				Spec: traefikIngressRouteTCPSpec{
					Routes: []traefikRouteTCP{
						{
							Match: "HostSNI(`*`)",
						},
					},
				},
			},
			expected: nil,
		},
	} {
		t.Run(ti.title, func(t *testing.T) {
			t.Parallel()

			fakeKubernetesClient := fakeKube.NewSimpleClientset()
			scheme := runtime.NewScheme()
			scheme.AddKnownTypes(ingressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(ingressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(ingressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			scheme.AddKnownTypes(oldIngressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(oldIngressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(oldIngressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			fakeDynamicClient := fakeDynamic.NewSimpleDynamicClient(scheme)

			ir := unstructured.Unstructured{}

			ingressRouteAsJSON, err := json.Marshal(ti.ingressRouteTCP)
			assert.NoError(t, err)

			assert.NoError(t, ir.UnmarshalJSON(ingressRouteAsJSON))

			// Create proxy resources
			_, err = fakeDynamicClient.Resource(oldIngressRouteTCPGVR).Namespace(defaultTraefikNamespace).Create(context.Background(), &ir, metav1.CreateOptions{})
			assert.NoError(t, err)

			source, err := NewTraefikSource(context.TODO(), fakeDynamicClient, fakeKubernetesClient, defaultTraefikNamespace, "kubernetes.io/ingress.class=traefik", ti.ignoreHostnameAnnotation, true, false)
			assert.NoError(t, err)
			assert.NotNil(t, source)

			count := &unstructured.UnstructuredList{}
			for len(count.Items) < 1 {
				count, _ = fakeDynamicClient.Resource(oldIngressRouteTCPGVR).Namespace(defaultTraefikNamespace).List(context.Background(), metav1.ListOptions{})
			}

			endpoints, err := source.Endpoints(context.Background())
			assert.NoError(t, err)
			assert.Len(t, endpoints, len(ti.expected))
			assert.Equal(t, ti.expected, endpoints)
		})
	}
}

func TestTraefikProxyOldIngressRouteUDPEndpoints(t *testing.T) {
	t.Parallel()

	for _, ti := range []struct {
		title                    string
		ingressRouteUDP          IngressRouteUDP
		ignoreHostnameAnnotation bool
		expected                 []*endpoint.Endpoint
	}{
		{
			title: "IngressRouteTCP with hostname annotation",
			ingressRouteUDP: IngressRouteUDP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteUDPGVR.GroupVersion().String(),
					Kind:       "IngressRouteUDP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressrouteudp-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "a.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressrouteudp/traefik/ingressrouteudp-annotation",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP with multiple hostname annotation",
			ingressRouteUDP: IngressRouteUDP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteUDPGVR.GroupVersion().String(),
					Kind:       "IngressRouteUDP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressrouteudp-multi-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com, b.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "a.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressrouteudp/traefik/ingressrouteudp-multi-annotation",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
				{
					DNSName:    "b.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressrouteudp/traefik/ingressrouteudp-multi-annotation",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRouteTCP ignoring hostname annotation",
			ingressRouteUDP: IngressRouteUDP{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteUDPGVR.GroupVersion().String(),
					Kind:       "IngressRouteUDP",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressrouteudp-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			ignoreHostnameAnnotation: true,
			expected:                 nil,
		},
	} {
		t.Run(ti.title, func(t *testing.T) {
			t.Parallel()

			fakeKubernetesClient := fakeKube.NewSimpleClientset()
			scheme := runtime.NewScheme()
			scheme.AddKnownTypes(ingressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(ingressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(ingressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			scheme.AddKnownTypes(oldIngressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(oldIngressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(oldIngressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			fakeDynamicClient := fakeDynamic.NewSimpleDynamicClient(scheme)

			ir := unstructured.Unstructured{}

			ingressRouteAsJSON, err := json.Marshal(ti.ingressRouteUDP)
			assert.NoError(t, err)

			assert.NoError(t, ir.UnmarshalJSON(ingressRouteAsJSON))

			// Create proxy resources
			_, err = fakeDynamicClient.Resource(oldIngressRouteUDPGVR).Namespace(defaultTraefikNamespace).Create(context.Background(), &ir, metav1.CreateOptions{})
			assert.NoError(t, err)

			source, err := NewTraefikSource(context.TODO(), fakeDynamicClient, fakeKubernetesClient, defaultTraefikNamespace, "kubernetes.io/ingress.class=traefik", ti.ignoreHostnameAnnotation, true, false)
			assert.NoError(t, err)
			assert.NotNil(t, source)

			count := &unstructured.UnstructuredList{}
			for len(count.Items) < 1 {
				count, _ = fakeDynamicClient.Resource(oldIngressRouteUDPGVR).Namespace(defaultTraefikNamespace).List(context.Background(), metav1.ListOptions{})
			}

			endpoints, err := source.Endpoints(context.Background())
			assert.NoError(t, err)
			assert.Len(t, endpoints, len(ti.expected))
			assert.Equal(t, ti.expected, endpoints)
		})
	}
}

func TestTraefikAPIGroupFlags(t *testing.T) {
	t.Parallel()

	for _, ti := range []struct {
		title                    string
		ingressRoute             IngressRoute
		gvr                      schema.GroupVersionResource
		ignoreHostnameAnnotation bool
		enableLegacy             bool
		disableNew               bool
		expected                 []*endpoint.Endpoint
	}{
		{
			title: "IngressRoute.traefik.containo.us with the legacy API group enabled",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			gvr:          oldIngressRouteGVR,
			enableLegacy: true,
			disableNew:   false,
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "a.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-annotation",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute.traefik.containo.us with the legacy API group disabled",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: oldIngressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			gvr:          oldIngressRouteGVR,
			enableLegacy: false,
			disableNew:   false,
		},
		{
			title: "IngressRoute.traefik.io with the new API group enabled",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			gvr:          ingressRouteGVR,
			enableLegacy: true,
			disableNew:   false,
			expected: []*endpoint.Endpoint{
				{
					DNSName:    "a.example.com",
					Targets:    []string{"target.domain.tld"},
					RecordType: endpoint.RecordTypeCNAME,
					RecordTTL:  0,
					Labels: endpoint.Labels{
						"resource": "ingressroute/traefik/ingressroute-annotation",
					},
					ProviderSpecific: endpoint.ProviderSpecific{},
				},
			},
		},
		{
			title: "IngressRoute.traefik.io with the new API group disabled",
			ingressRoute: IngressRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingressRouteGVR.GroupVersion().String(),
					Kind:       "IngressRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute-annotation",
					Namespace: defaultTraefikNamespace,
					Annotations: map[string]string{
						"external-dns.alpha.kubernetes.io/hostname": "a.example.com",
						"external-dns.alpha.kubernetes.io/target":   "target.domain.tld",
						"kubernetes.io/ingress.class":               "traefik",
					},
				},
			},
			gvr:          ingressRouteGVR,
			enableLegacy: true,
			disableNew:   true,
		},
	} {
		t.Run(ti.title, func(t *testing.T) {
			t.Parallel()

			fakeKubernetesClient := fakeKube.NewSimpleClientset()
			scheme := runtime.NewScheme()
			scheme.AddKnownTypes(ingressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(ingressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(ingressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			scheme.AddKnownTypes(oldIngressRouteGVR.GroupVersion(), &IngressRoute{}, &IngressRouteList{})
			scheme.AddKnownTypes(oldIngressRouteTCPGVR.GroupVersion(), &IngressRouteTCP{}, &IngressRouteTCPList{})
			scheme.AddKnownTypes(oldIngressRouteUDPGVR.GroupVersion(), &IngressRouteUDP{}, &IngressRouteUDPList{})
			fakeDynamicClient := fakeDynamic.NewSimpleDynamicClient(scheme)

			ir := unstructured.Unstructured{}

			ingressRouteAsJSON, err := json.Marshal(ti.ingressRoute)
			assert.NoError(t, err)

			assert.NoError(t, ir.UnmarshalJSON(ingressRouteAsJSON))

			// Create proxy resources
			_, err = fakeDynamicClient.Resource(ti.gvr).Namespace(defaultTraefikNamespace).Create(context.Background(), &ir, metav1.CreateOptions{})
			assert.NoError(t, err)

			source, err := NewTraefikSource(context.TODO(), fakeDynamicClient, fakeKubernetesClient, defaultTraefikNamespace, "kubernetes.io/ingress.class=traefik", ti.ignoreHostnameAnnotation, ti.enableLegacy, ti.disableNew)
			assert.NoError(t, err)
			assert.NotNil(t, source)

			count := &unstructured.UnstructuredList{}
			for len(count.Items) < 1 {
				count, _ = fakeDynamicClient.Resource(ti.gvr).Namespace(defaultTraefikNamespace).List(context.Background(), metav1.ListOptions{})
			}

			endpoints, err := source.Endpoints(context.Background())
			assert.NoError(t, err)
			assert.Len(t, endpoints, len(ti.expected))
			assert.Equal(t, ti.expected, endpoints)
		})
	}
}

func TestAddEventHandler_AllBranches(t *testing.T) {
	ctx := context.Background()
	handlerCalled := false
	handler := func() { handlerCalled = true }

	inf := testInformer{}
	fakeInformer := new(FakeInformer)
	fakeInformer.On("Informer").Return(&inf)

	cases := []struct {
		name string
		ts   *traefikSource
		want int
	}{
		{"all nil", &traefikSource{}, 0},
		{"all set", &traefikSource{
			ingressRouteInformer:       fakeInformer,
			oldIngressRouteInformer:    fakeInformer,
			ingressRouteTcpInformer:    fakeInformer,
			oldIngressRouteTcpInformer: fakeInformer,
			ingressRouteUdpInformer:    fakeInformer,
			oldIngressRouteUdpInformer: fakeInformer,
		}, 6},
		{"some set", &traefikSource{
			ingressRouteInformer:       fakeInformer,
			oldIngressRouteInformer:    fakeInformer,
			ingressRouteTcpInformer:    nil,
			oldIngressRouteTcpInformer: fakeInformer,
			ingressRouteUdpInformer:    nil,
			oldIngressRouteUdpInformer: nil,
		}, 3},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			test.ts.AddEventHandler(ctx, handler)
			assert.Equal(t, test.want, inf.times)
			assert.False(t, handlerCalled)

			if test.want > 0 {
				fakeInformer.AssertExpectations(t)
				fakeInformer.AssertCalled(t, "Informer")
			} else {
				fakeInformer.AssertNotCalled(t, "Informer")
			}
			// reset the call count
			inf.times = 0
		})
	}
}

type FakeInformer struct {
	mock.Mock
	informer cache.SharedIndexInformer
	lister   cache.GenericLister
}

func (f *FakeInformer) Informer() cache.SharedIndexInformer {
	args := f.Called()
	return args.Get(0).(cache.SharedIndexInformer)
}

func (f *FakeInformer) Lister() cache.GenericLister {
	return f.lister
}

type testInformer struct {
	cache.SharedIndexInformer

	times int
}

func (t *testInformer) AddEventHandler(_ cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	t.times += 1
	return nil, fmt.Errorf("not implemented")
}

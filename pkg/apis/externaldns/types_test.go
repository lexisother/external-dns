/*
Copyright 2017 The Kubernetes Authors.

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

package externaldns

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/external-dns/endpoint"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	minimalConfig = &Config{
		APIServerURL:                           "",
		KubeConfig:                             "",
		RequestTimeout:                         time.Second * 30,
		GlooNamespaces:                         []string{"gloo-system"},
		SkipperRouteGroupVersion:               "zalando.org/v1",
		Sources:                                []string{"service"},
		Namespace:                              "",
		FQDNTemplate:                           "",
		Compatibility:                          "",
		Provider:                               "google",
		GoogleProject:                          "",
		GoogleBatchChangeSize:                  1000,
		GoogleBatchChangeInterval:              time.Second,
		GoogleZoneVisibility:                   "",
		DomainFilter:                           []string{""},
		ExcludeDomains:                         []string{""},
		RegexDomainFilter:                      regexp.MustCompile(""),
		RegexDomainExclusion:                   regexp.MustCompile(""),
		ZoneNameFilter:                         []string{""},
		ZoneIDFilter:                           []string{""},
		AlibabaCloudConfigFile:                 "/etc/kubernetes/alibaba-cloud.json",
		AWSZoneType:                            "",
		AWSZoneTagFilter:                       []string{""},
		AWSZoneMatchParent:                     false,
		AWSAssumeRole:                          "",
		AWSAssumeRoleExternalID:                "",
		AWSBatchChangeSize:                     1000,
		AWSBatchChangeSizeBytes:                32000,
		AWSBatchChangeSizeValues:               1000,
		AWSBatchChangeInterval:                 time.Second,
		AWSEvaluateTargetHealth:                true,
		AWSAPIRetries:                          3,
		AWSPreferCNAME:                         false,
		AWSProfiles:                            []string{""},
		AWSZoneCacheDuration:                   0 * time.Second,
		AWSSDServiceCleanup:                    false,
		AWSSDCreateTag:                         map[string]string{},
		AWSDynamoDBTable:                       "external-dns",
		AzureConfigFile:                        "/etc/kubernetes/azure.json",
		AzureResourceGroup:                     "",
		AzureSubscriptionID:                    "",
		CloudflareProxied:                      false,
		CloudflareCustomHostnames:              false,
		CloudflareCustomHostnamesMinTLSVersion: "1.0",
		CloudflareCustomHostnamesCertificateAuthority: "google",
		CloudflareDNSRecordsPerPage:                   100,
		CloudflareRegionKey:                           "",
		CoreDNSPrefix:                                 "/skydns/",
		AkamaiServiceConsumerDomain:                   "",
		AkamaiClientToken:                             "",
		AkamaiClientSecret:                            "",
		AkamaiAccessToken:                             "",
		AkamaiEdgercPath:                              "",
		AkamaiEdgercSection:                           "",
		OCIConfigFile:                                 "/etc/kubernetes/oci.yaml",
		OCIZoneScope:                                  "GLOBAL",
		OCIZoneCacheDuration:                          0 * time.Second,
		InMemoryZones:                                 []string{""},
		OVHEndpoint:                                   "ovh-eu",
		OVHApiRateLimit:                               20,
		PDNSServer:                                    "http://localhost:8081",
		PDNSServerID:                                  "localhost",
		PDNSAPIKey:                                    "",
		Policy:                                        "sync",
		Registry:                                      "txt",
		TXTOwnerID:                                    "default",
		TXTPrefix:                                     "",
		TXTCacheInterval:                              0,
		TXTNewFormatOnly:                              false,
		Interval:                                      time.Minute,
		MinEventSyncInterval:                          5 * time.Second,
		Once:                                          false,
		DryRun:                                        false,
		UpdateEvents:                                  false,
		LogFormat:                                     "text",
		MetricsAddress:                                ":7979",
		LogLevel:                                      logrus.InfoLevel.String(),
		ConnectorSourceServer:                         "localhost:8080",
		ExoscaleAPIEnvironment:                        "api",
		ExoscaleAPIZone:                               "ch-gva-2",
		ExoscaleAPIKey:                                "",
		ExoscaleAPISecret:                             "",
		CRDSourceAPIVersion:                           "externaldns.k8s.io/v1alpha1",
		CRDSourceKind:                                 "DNSEndpoint",
		TransIPAccountName:                            "",
		TransIPPrivateKeyFile:                         "",
		DigitalOceanAPIPageSize:                       50,
		ManagedDNSRecordTypes:                         []string{endpoint.RecordTypeA, endpoint.RecordTypeAAAA, endpoint.RecordTypeCNAME},
		RFC2136BatchChangeSize:                        50,
		RFC2136Host:                                   []string{""},
		RFC2136LoadBalancingStrategy:                  "disabled",
		OCPRouterName:                                 "default",
		IBMCloudProxied:                               false,
		IBMCloudConfigFile:                            "/etc/kubernetes/ibmcloud.json",
		TencentCloudConfigFile:                        "/etc/kubernetes/tencent-cloud.json",
		TencentCloudZoneType:                          "",
		WebhookProviderURL:                            "http://localhost:8888",
		WebhookProviderReadTimeout:                    5 * time.Second,
		WebhookProviderWriteTimeout:                   10 * time.Second,
	}

	overriddenConfig = &Config{
		APIServerURL:                           "http://127.0.0.1:8080",
		KubeConfig:                             "/some/path",
		RequestTimeout:                         time.Second * 77,
		GlooNamespaces:                         []string{"gloo-not-system", "gloo-second-system"},
		SkipperRouteGroupVersion:               "zalando.org/v2",
		Sources:                                []string{"service", "ingress", "connector"},
		Namespace:                              "namespace",
		IgnoreHostnameAnnotation:               true,
		IgnoreNonHostNetworkPods:               false,
		IgnoreIngressTLSSpec:                   true,
		IgnoreIngressRulesSpec:                 true,
		FQDNTemplate:                           "{{.Name}}.service.example.com",
		Compatibility:                          "mate",
		Provider:                               "google",
		GoogleProject:                          "project",
		GoogleBatchChangeSize:                  100,
		GoogleBatchChangeInterval:              time.Second * 2,
		GoogleZoneVisibility:                   "private",
		DomainFilter:                           []string{"example.org", "company.com"},
		ExcludeDomains:                         []string{"xapi.example.org", "xapi.company.com"},
		RegexDomainFilter:                      regexp.MustCompile("(example\\.org|company\\.com)$"),
		RegexDomainExclusion:                   regexp.MustCompile("xapi\\.(example\\.org|company\\.com)$"),
		ZoneNameFilter:                         []string{"yapi.example.org", "yapi.company.com"},
		ZoneIDFilter:                           []string{"/hostedzone/ZTST1", "/hostedzone/ZTST2"},
		TargetNetFilter:                        []string{"10.0.0.0/9", "10.1.0.0/9"},
		ExcludeTargetNets:                      []string{"1.0.0.0/9", "1.1.0.0/9"},
		AlibabaCloudConfigFile:                 "/etc/kubernetes/alibaba-cloud.json",
		AWSZoneType:                            "private",
		AWSZoneTagFilter:                       []string{"tag=foo"},
		AWSZoneMatchParent:                     true,
		AWSAssumeRole:                          "some-other-role",
		AWSAssumeRoleExternalID:                "pg2000",
		AWSBatchChangeSize:                     100,
		AWSBatchChangeSizeBytes:                16000,
		AWSBatchChangeSizeValues:               100,
		AWSBatchChangeInterval:                 time.Second * 2,
		AWSEvaluateTargetHealth:                false,
		AWSAPIRetries:                          13,
		AWSPreferCNAME:                         true,
		AWSProfiles:                            []string{"profile1", "profile2"},
		AWSZoneCacheDuration:                   10 * time.Second,
		AWSSDServiceCleanup:                    true,
		AWSSDCreateTag:                         map[string]string{"key1": "value1", "key2": "value2"},
		AWSDynamoDBTable:                       "custom-table",
		AzureConfigFile:                        "azure.json",
		AzureResourceGroup:                     "arg",
		AzureSubscriptionID:                    "arg",
		CloudflareProxied:                      true,
		CloudflareCustomHostnames:              true,
		CloudflareCustomHostnamesMinTLSVersion: "1.3",
		CloudflareCustomHostnamesCertificateAuthority: "google",
		CloudflareDNSRecordsPerPage:                   5000,
		CloudflareRegionKey:                           "us",
		CoreDNSPrefix:                                 "/coredns/",
		AkamaiServiceConsumerDomain:                   "oooo-xxxxxxxxxxxxxxxx-xxxxxxxxxxxxxxxx.luna.akamaiapis.net",
		AkamaiClientToken:                             "o184671d5307a388180fbf7f11dbdf46",
		AkamaiClientSecret:                            "o184671d5307a388180fbf7f11dbdf46",
		AkamaiAccessToken:                             "o184671d5307a388180fbf7f11dbdf46",
		AkamaiEdgercPath:                              "/home/test/.edgerc",
		AkamaiEdgercSection:                           "default",
		OCIConfigFile:                                 "oci.yaml",
		OCIZoneScope:                                  "PRIVATE",
		OCIZoneCacheDuration:                          30 * time.Second,
		InMemoryZones:                                 []string{"example.org", "company.com"},
		OVHEndpoint:                                   "ovh-ca",
		OVHApiRateLimit:                               42,
		PDNSServer:                                    "http://ns.example.com:8081",
		PDNSServerID:                                  "localhost",
		PDNSAPIKey:                                    "some-secret-key",
		PDNSSkipTLSVerify:                             true,
		TLSCA:                                         "/path/to/ca.crt",
		TLSClientCert:                                 "/path/to/cert.pem",
		TLSClientCertKey:                              "/path/to/key.pem",
		PodSourceDomain:                               "example.org",
		Policy:                                        "upsert-only",
		Registry:                                      "noop",
		TXTOwnerID:                                    "owner-1",
		TXTPrefix:                                     "associated-txt-record",
		TXTCacheInterval:                              12 * time.Hour,
		TXTNewFormatOnly:                              true,
		Interval:                                      10 * time.Minute,
		MinEventSyncInterval:                          50 * time.Second,
		Once:                                          true,
		DryRun:                                        true,
		UpdateEvents:                                  true,
		LogFormat:                                     "json",
		MetricsAddress:                                "127.0.0.1:9099",
		LogLevel:                                      logrus.DebugLevel.String(),
		ConnectorSourceServer:                         "localhost:8081",
		ExoscaleAPIEnvironment:                        "api1",
		ExoscaleAPIZone:                               "zone1",
		ExoscaleAPIKey:                                "1",
		ExoscaleAPISecret:                             "2",
		CRDSourceAPIVersion:                           "test.k8s.io/v1alpha1",
		CRDSourceKind:                                 "Endpoint",
		NS1Endpoint:                                   "https://api.example.com/v1",
		NS1IgnoreSSL:                                  true,
		TransIPAccountName:                            "transip",
		TransIPPrivateKeyFile:                         "/path/to/transip.key",
		DigitalOceanAPIPageSize:                       100,
		ManagedDNSRecordTypes:                         []string{endpoint.RecordTypeA, endpoint.RecordTypeAAAA, endpoint.RecordTypeCNAME, endpoint.RecordTypeNS},
		RFC2136BatchChangeSize:                        100,
		RFC2136Host:                                   []string{"rfc2136-host1", "rfc2136-host2"},
		RFC2136LoadBalancingStrategy:                  "round-robin",
		IBMCloudProxied:                               true,
		IBMCloudConfigFile:                            "ibmcloud.json",
		TencentCloudConfigFile:                        "tencent-cloud.json",
		TencentCloudZoneType:                          "private",
		WebhookProviderURL:                            "http://localhost:8888",
		WebhookProviderReadTimeout:                    5 * time.Second,
		WebhookProviderWriteTimeout:                   10 * time.Second,
	}
)

func TestParseFlags(t *testing.T) {
	for _, ti := range []struct {
		title    string
		args     []string
		envVars  map[string]string
		expected *Config
	}{
		{
			title: "default config with minimal flags defined",
			args: []string{
				"--source=service",
				"--provider=google",
				"--openshift-router-name=default",
			},
			envVars:  map[string]string{},
			expected: minimalConfig,
		},
		{
			title: "override everything via flags",
			args: []string{
				"--server=http://127.0.0.1:8080",
				"--kubeconfig=/some/path",
				"--request-timeout=77s",
				"--gloo-namespace=gloo-not-system",
				"--gloo-namespace=gloo-second-system",
				"--skipper-routegroup-groupversion=zalando.org/v2",
				"--source=service",
				"--source=ingress",
				"--source=connector",
				"--namespace=namespace",
				"--fqdn-template={{.Name}}.service.example.com",
				"--no-ignore-non-host-network-pods",
				"--ignore-hostname-annotation",
				"--ignore-ingress-tls-spec",
				"--ignore-ingress-rules-spec",
				"--compatibility=mate",
				"--provider=google",
				"--google-project=project",
				"--google-batch-change-size=100",
				"--google-batch-change-interval=2s",
				"--google-zone-visibility=private",
				"--azure-config-file=azure.json",
				"--azure-resource-group=arg",
				"--azure-subscription-id=arg",
				"--cloudflare-proxied",
				"--cloudflare-custom-hostnames",
				"--cloudflare-custom-hostnames-min-tls-version=1.3",
				"--cloudflare-custom-hostnames-certificate-authority=google",
				"--cloudflare-dns-records-per-page=5000",
				"--cloudflare-region-key=us",
				"--coredns-prefix=/coredns/",
				"--akamai-serviceconsumerdomain=oooo-xxxxxxxxxxxxxxxx-xxxxxxxxxxxxxxxx.luna.akamaiapis.net",
				"--akamai-client-token=o184671d5307a388180fbf7f11dbdf46",
				"--akamai-client-secret=o184671d5307a388180fbf7f11dbdf46",
				"--akamai-access-token=o184671d5307a388180fbf7f11dbdf46",
				"--akamai-edgerc-path=/home/test/.edgerc",
				"--akamai-edgerc-section=default",
				"--inmemory-zone=example.org",
				"--inmemory-zone=company.com",
				"--ovh-endpoint=ovh-ca",
				"--ovh-api-rate-limit=42",
				"--pdns-server=http://ns.example.com:8081",
				"--pdns-server-id=localhost",
				"--pdns-api-key=some-secret-key",
				"--pdns-skip-tls-verify",
				"--oci-config-file=oci.yaml",
				"--oci-zone-scope=PRIVATE",
				"--oci-zones-cache-duration=30s",
				"--tls-ca=/path/to/ca.crt",
				"--tls-client-cert=/path/to/cert.pem",
				"--tls-client-cert-key=/path/to/key.pem",
				"--pod-source-domain=example.org",
				"--domain-filter=example.org",
				"--domain-filter=company.com",
				"--exclude-domains=xapi.example.org",
				"--exclude-domains=xapi.company.com",
				"--regex-domain-filter=(example\\.org|company\\.com)$",
				"--regex-domain-exclusion=xapi\\.(example\\.org|company\\.com)$",
				"--zone-name-filter=yapi.example.org",
				"--zone-name-filter=yapi.company.com",
				"--zone-id-filter=/hostedzone/ZTST1",
				"--zone-id-filter=/hostedzone/ZTST2",
				"--target-net-filter=10.0.0.0/9",
				"--target-net-filter=10.1.0.0/9",
				"--exclude-target-net=1.0.0.0/9",
				"--exclude-target-net=1.1.0.0/9",
				"--aws-zone-type=private",
				"--aws-zone-tags=tag=foo",
				"--aws-zone-match-parent",
				"--aws-assume-role=some-other-role",
				"--aws-assume-role-external-id=pg2000",
				"--aws-batch-change-size=100",
				"--aws-batch-change-size-bytes=16000",
				"--aws-batch-change-size-values=100",
				"--aws-batch-change-interval=2s",
				"--aws-api-retries=13",
				"--aws-prefer-cname",
				"--aws-profile=profile1",
				"--aws-profile=profile2",
				"--aws-zones-cache-duration=10s",
				"--aws-sd-service-cleanup",
				"--aws-sd-create-tag=key1=value1",
				"--aws-sd-create-tag=key2=value2",
				"--no-aws-evaluate-target-health",
				"--policy=upsert-only",
				"--registry=noop",
				"--txt-owner-id=owner-1",
				"--txt-prefix=associated-txt-record",
				"--txt-cache-interval=12h",
				"--txt-new-format-only",
				"--dynamodb-table=custom-table",
				"--interval=10m",
				"--min-event-sync-interval=50s",
				"--once",
				"--dry-run",
				"--events",
				"--log-format=json",
				"--metrics-address=127.0.0.1:9099",
				"--log-level=debug",
				"--connector-source-server=localhost:8081",
				"--exoscale-apienv=api1",
				"--exoscale-apizone=zone1",
				"--exoscale-apikey=1",
				"--exoscale-apisecret=2",
				"--crd-source-apiversion=test.k8s.io/v1alpha1",
				"--crd-source-kind=Endpoint",
				"--ns1-endpoint=https://api.example.com/v1",
				"--ns1-ignoressl",
				"--transip-account=transip",
				"--transip-keyfile=/path/to/transip.key",
				"--digitalocean-api-page-size=100",
				"--managed-record-types=A",
				"--managed-record-types=AAAA",
				"--managed-record-types=CNAME",
				"--managed-record-types=NS",
				"--rfc2136-batch-change-size=100",
				"--rfc2136-load-balancing-strategy=round-robin",
				"--rfc2136-host=rfc2136-host1",
				"--rfc2136-host=rfc2136-host2",
				"--ibmcloud-proxied",
				"--ibmcloud-config-file=ibmcloud.json",
				"--tencent-cloud-config-file=tencent-cloud.json",
				"--tencent-cloud-zone-type=private",
			},
			envVars:  map[string]string{},
			expected: overriddenConfig,
		},
		{
			title: "override everything via environment variables",
			args:  []string{},
			envVars: map[string]string{
				"EXTERNAL_DNS_SERVER":                                            "http://127.0.0.1:8080",
				"EXTERNAL_DNS_KUBECONFIG":                                        "/some/path",
				"EXTERNAL_DNS_REQUEST_TIMEOUT":                                   "77s",
				"EXTERNAL_DNS_CONTOUR_LOAD_BALANCER":                             "heptio-contour-other/contour-other",
				"EXTERNAL_DNS_GLOO_NAMESPACE":                                    "gloo-not-system\ngloo-second-system",
				"EXTERNAL_DNS_SKIPPER_ROUTEGROUP_GROUPVERSION":                   "zalando.org/v2",
				"EXTERNAL_DNS_SOURCE":                                            "service\ningress\nconnector",
				"EXTERNAL_DNS_NAMESPACE":                                         "namespace",
				"EXTERNAL_DNS_FQDN_TEMPLATE":                                     "{{.Name}}.service.example.com",
				"EXTERNAL_DNS_IGNORE_NON_HOST_NETWORK_PODS":                      "0",
				"EXTERNAL_DNS_IGNORE_HOSTNAME_ANNOTATION":                        "1",
				"EXTERNAL_DNS_IGNORE_INGRESS_TLS_SPEC":                           "1",
				"EXTERNAL_DNS_IGNORE_INGRESS_RULES_SPEC":                         "1",
				"EXTERNAL_DNS_COMPATIBILITY":                                     "mate",
				"EXTERNAL_DNS_PROVIDER":                                          "google",
				"EXTERNAL_DNS_GOOGLE_PROJECT":                                    "project",
				"EXTERNAL_DNS_GOOGLE_BATCH_CHANGE_SIZE":                          "100",
				"EXTERNAL_DNS_GOOGLE_BATCH_CHANGE_INTERVAL":                      "2s",
				"EXTERNAL_DNS_GOOGLE_ZONE_VISIBILITY":                            "private",
				"EXTERNAL_DNS_AZURE_CONFIG_FILE":                                 "azure.json",
				"EXTERNAL_DNS_AZURE_RESOURCE_GROUP":                              "arg",
				"EXTERNAL_DNS_AZURE_SUBSCRIPTION_ID":                             "arg",
				"EXTERNAL_DNS_CLOUDFLARE_PROXIED":                                "1",
				"EXTERNAL_DNS_CLOUDFLARE_CUSTOM_HOSTNAMES":                       "1",
				"EXTERNAL_DNS_CLOUDFLARE_CUSTOM_HOSTNAMES_MIN_TLS_VERSION":       "1.3",
				"EXTERNAL_DNS_CLOUDFLARE_CUSTOM_HOSTNAMES_CERTIFICATE_AUTHORITY": "google",
				"EXTERNAL_DNS_CLOUDFLARE_DNS_RECORDS_PER_PAGE":                   "5000",
				"EXTERNAL_DNS_CLOUDFLARE_REGION_KEY":                             "us",
				"EXTERNAL_DNS_COREDNS_PREFIX":                                    "/coredns/",
				"EXTERNAL_DNS_AKAMAI_SERVICECONSUMERDOMAIN":                      "oooo-xxxxxxxxxxxxxxxx-xxxxxxxxxxxxxxxx.luna.akamaiapis.net",
				"EXTERNAL_DNS_AKAMAI_CLIENT_TOKEN":                               "o184671d5307a388180fbf7f11dbdf46",
				"EXTERNAL_DNS_AKAMAI_CLIENT_SECRET":                              "o184671d5307a388180fbf7f11dbdf46",
				"EXTERNAL_DNS_AKAMAI_ACCESS_TOKEN":                               "o184671d5307a388180fbf7f11dbdf46",
				"EXTERNAL_DNS_AKAMAI_EDGERC_PATH":                                "/home/test/.edgerc",
				"EXTERNAL_DNS_AKAMAI_EDGERC_SECTION":                             "default",
				"EXTERNAL_DNS_OCI_CONFIG_FILE":                                   "oci.yaml",
				"EXTERNAL_DNS_OCI_ZONE_SCOPE":                                    "PRIVATE",
				"EXTERNAL_DNS_OCI_ZONES_CACHE_DURATION":                          "30s",
				"EXTERNAL_DNS_INMEMORY_ZONE":                                     "example.org\ncompany.com",
				"EXTERNAL_DNS_OVH_ENDPOINT":                                      "ovh-ca",
				"EXTERNAL_DNS_OVH_API_RATE_LIMIT":                                "42",
				"EXTERNAL_DNS_POD_SOURCE_DOMAIN":                                 "example.org",
				"EXTERNAL_DNS_DOMAIN_FILTER":                                     "example.org\ncompany.com",
				"EXTERNAL_DNS_EXCLUDE_DOMAINS":                                   "xapi.example.org\nxapi.company.com",
				"EXTERNAL_DNS_REGEX_DOMAIN_FILTER":                               "(example\\.org|company\\.com)$",
				"EXTERNAL_DNS_REGEX_DOMAIN_EXCLUSION":                            "xapi\\.(example\\.org|company\\.com)$",
				"EXTERNAL_DNS_TARGET_NET_FILTER":                                 "10.0.0.0/9\n10.1.0.0/9",
				"EXTERNAL_DNS_EXCLUDE_TARGET_NET":                                "1.0.0.0/9\n1.1.0.0/9",
				"EXTERNAL_DNS_PDNS_SERVER":                                       "http://ns.example.com:8081",
				"EXTERNAL_DNS_PDNS_ID":                                           "localhost",
				"EXTERNAL_DNS_PDNS_API_KEY":                                      "some-secret-key",
				"EXTERNAL_DNS_PDNS_SKIP_TLS_VERIFY":                              "1",
				"EXTERNAL_DNS_RDNS_ROOT_DOMAIN":                                  "lb.rancher.cloud",
				"EXTERNAL_DNS_TLS_CA":                                            "/path/to/ca.crt",
				"EXTERNAL_DNS_TLS_CLIENT_CERT":                                   "/path/to/cert.pem",
				"EXTERNAL_DNS_TLS_CLIENT_CERT_KEY":                               "/path/to/key.pem",
				"EXTERNAL_DNS_ZONE_NAME_FILTER":                                  "yapi.example.org\nyapi.company.com",
				"EXTERNAL_DNS_ZONE_ID_FILTER":                                    "/hostedzone/ZTST1\n/hostedzone/ZTST2",
				"EXTERNAL_DNS_AWS_ZONE_TYPE":                                     "private",
				"EXTERNAL_DNS_AWS_ZONE_TAGS":                                     "tag=foo",
				"EXTERNAL_DNS_AWS_ZONE_MATCH_PARENT":                             "true",
				"EXTERNAL_DNS_AWS_ASSUME_ROLE":                                   "some-other-role",
				"EXTERNAL_DNS_AWS_ASSUME_ROLE_EXTERNAL_ID":                       "pg2000",
				"EXTERNAL_DNS_AWS_BATCH_CHANGE_SIZE":                             "100",
				"EXTERNAL_DNS_AWS_BATCH_CHANGE_SIZE_BYTES":                       "16000",
				"EXTERNAL_DNS_AWS_BATCH_CHANGE_SIZE_VALUES":                      "100",
				"EXTERNAL_DNS_AWS_BATCH_CHANGE_INTERVAL":                         "2s",
				"EXTERNAL_DNS_AWS_EVALUATE_TARGET_HEALTH":                        "0",
				"EXTERNAL_DNS_AWS_API_RETRIES":                                   "13",
				"EXTERNAL_DNS_AWS_PREFER_CNAME":                                  "true",
				"EXTERNAL_DNS_AWS_PROFILE":                                       "profile1\nprofile2",
				"EXTERNAL_DNS_AWS_ZONES_CACHE_DURATION":                          "10s",
				"EXTERNAL_DNS_AWS_SD_SERVICE_CLEANUP":                            "true",
				"EXTERNAL_DNS_AWS_SD_CREATE_TAG":                                 "key1=value1\nkey2=value2",
				"EXTERNAL_DNS_DYNAMODB_TABLE":                                    "custom-table",
				"EXTERNAL_DNS_POLICY":                                            "upsert-only",
				"EXTERNAL_DNS_REGISTRY":                                          "noop",
				"EXTERNAL_DNS_TXT_OWNER_ID":                                      "owner-1",
				"EXTERNAL_DNS_TXT_PREFIX":                                        "associated-txt-record",
				"EXTERNAL_DNS_TXT_CACHE_INTERVAL":                                "12h",
				"EXTERNAL_DNS_TXT_NEW_FORMAT_ONLY":                               "1",
				"EXTERNAL_DNS_INTERVAL":                                          "10m",
				"EXTERNAL_DNS_MIN_EVENT_SYNC_INTERVAL":                           "50s",
				"EXTERNAL_DNS_ONCE":                                              "1",
				"EXTERNAL_DNS_DRY_RUN":                                           "1",
				"EXTERNAL_DNS_EVENTS":                                            "1",
				"EXTERNAL_DNS_LOG_FORMAT":                                        "json",
				"EXTERNAL_DNS_METRICS_ADDRESS":                                   "127.0.0.1:9099",
				"EXTERNAL_DNS_LOG_LEVEL":                                         "debug",
				"EXTERNAL_DNS_CONNECTOR_SOURCE_SERVER":                           "localhost:8081",
				"EXTERNAL_DNS_EXOSCALE_APIENV":                                   "api1",
				"EXTERNAL_DNS_EXOSCALE_APIZONE":                                  "zone1",
				"EXTERNAL_DNS_EXOSCALE_APIKEY":                                   "1",
				"EXTERNAL_DNS_EXOSCALE_APISECRET":                                "2",
				"EXTERNAL_DNS_CRD_SOURCE_APIVERSION":                             "test.k8s.io/v1alpha1",
				"EXTERNAL_DNS_CRD_SOURCE_KIND":                                   "Endpoint",
				"EXTERNAL_DNS_NS1_ENDPOINT":                                      "https://api.example.com/v1",
				"EXTERNAL_DNS_NS1_IGNORESSL":                                     "1",
				"EXTERNAL_DNS_TRANSIP_ACCOUNT":                                   "transip",
				"EXTERNAL_DNS_TRANSIP_KEYFILE":                                   "/path/to/transip.key",
				"EXTERNAL_DNS_DIGITALOCEAN_API_PAGE_SIZE":                        "100",
				"EXTERNAL_DNS_MANAGED_RECORD_TYPES":                              "A\nAAAA\nCNAME\nNS",
				"EXTERNAL_DNS_RFC2136_BATCH_CHANGE_SIZE":                         "100",
				"EXTERNAL_DNS_RFC2136_LOAD_BALANCING_STRATEGY":                   "round-robin",
				"EXTERNAL_DNS_RFC2136_HOST":                                      "rfc2136-host1\nrfc2136-host2",
				"EXTERNAL_DNS_IBMCLOUD_PROXIED":                                  "1",
				"EXTERNAL_DNS_IBMCLOUD_CONFIG_FILE":                              "ibmcloud.json",
				"EXTERNAL_DNS_TENCENT_CLOUD_CONFIG_FILE":                         "tencent-cloud.json",
				"EXTERNAL_DNS_TENCENT_CLOUD_ZONE_TYPE":                           "private",
			},
			expected: overriddenConfig,
		},
	} {
		t.Run(ti.title, func(t *testing.T) {
			originalEnv := setEnv(t, ti.envVars)
			defer func() { restoreEnv(t, originalEnv) }()

			cfg := NewConfig()
			require.NoError(t, cfg.ParseFlags(ti.args))
			assert.Equal(t, ti.expected, cfg)
		})
	}
}

// helper functions

func setEnv(t *testing.T, env map[string]string) map[string]string {
	originalEnv := map[string]string{}

	for k, v := range env {
		originalEnv[k] = os.Getenv(k)
		require.NoError(t, os.Setenv(k, v))
	}

	return originalEnv
}

func restoreEnv(t *testing.T, originalEnv map[string]string) {
	for k, v := range originalEnv {
		require.NoError(t, os.Setenv(k, v))
	}
}

func TestPasswordsNotLogged(t *testing.T) {
	cfg := Config{
		PDNSAPIKey:        "pdns-api-key",
		RFC2136TSIGSecret: "tsig-secret",
	}

	s := cfg.String()

	assert.False(t, strings.Contains(s, "pdns-api-key"))
	assert.False(t, strings.Contains(s, "tsig-secret"))
}

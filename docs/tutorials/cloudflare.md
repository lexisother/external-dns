# Cloudflare DNS

This tutorial describes how to setup ExternalDNS for usage within a Kubernetes cluster using Cloudflare DNS.

Make sure to use **>=0.4.2** version of ExternalDNS for this tutorial.

## CloudFlare SDK Migration Status

ExternalDNS is currently migrating from the legacy CloudFlare Go SDK v0 to the modern v4 SDK to improve performance, reliability, and access to newer CloudFlare features. The migration status is:

**✅ Fully migrated to v4 SDK:**

- Zone management (listing, filtering, pagination)
- Zone details retrieval (`GetZone`)
- Zone ID lookup by name (`ZoneIDByName`)
- Zone plan detection (fully v4 implementation)
- Regional services (data localization)

**🔄 Still using legacy v0 SDK:**

- DNS record management (create, update, delete records)
- Custom hostnames
- Proxied records

This mixed approach ensures continued functionality while gradually modernizing the codebase. Users should not experience any breaking changes during this transition.

### SDK Dependencies

ExternalDNS currently uses:

- **cloudflare-go v0.115.0+**: Legacy SDK for DNS records, custom hostnames, and proxied record features
- **cloudflare-go/v4 v4.6.0+**: Modern SDK for all zone management and regional services operations

Zone management has been fully migrated to the v4 SDK, providing improved performance and reliability.

Both SDKs are automatically managed as Go module dependencies and require no special configuration from users.

## Creating a Cloudflare DNS zone

We highly recommend to read this tutorial if you haven't used Cloudflare before:

[Create a Cloudflare account and add a website](https://support.cloudflare.com/hc/en-us/articles/201720164-Step-2-Create-a-Cloudflare-account-and-add-a-website)

## Creating Cloudflare Credentials

Snippet from [Cloudflare - Getting Started](https://api.cloudflare.com/#getting-started-endpoints):

> Cloudflare's API exposes the entire Cloudflare infrastructure via a standardized programmatic interface. Using Cloudflare's API, you can do just about anything you can do on cloudflare.com via the customer dashboard.
> The Cloudflare API is a RESTful API based on HTTPS requests and JSON responses. If you are registered with Cloudflare, you can obtain your API key from the bottom of the "My Account" page, found here: [Go to My account](https://dash.cloudflare.com/profile).

API Token will be preferred for authentication if `CF_API_TOKEN` environment variable is set.
Otherwise `CF_API_KEY` and `CF_API_EMAIL` should be set to run ExternalDNS with Cloudflare.
You may provide the Cloudflare API token through a file by setting the
`CF_API_TOKEN="file:/path/to/token"`.

Note. The `CF_API_KEY` and `CF_API_EMAIL` should not be present, if you are using a `CF_API_TOKEN`.

When using API Token authentication, the token should be granted Zone `Read`, DNS `Edit` privileges, and access to `All zones`.

If you would like to further restrict the API permissions to a specific zone (or zones), you also need to use the `--zone-id-filter` so that the underlying API requests only access the zones that you explicitly specify, as opposed to accessing all zones.

## Throttling

Cloudflare API has a [global rate limit of 1,200 requests per five minutes](https://developers.cloudflare.com/fundamentals/api/reference/limits/). Running several fast polling ExternalDNS instances in a given account can easily hit that limit.
The AWS Provider [docs](./aws.md#throttling) has some recommendations that can be followed here too, but in particular, consider passing `--cloudflare-dns-records-per-page` with a high value (maximum is 5,000).

## Deploy ExternalDNS

Connect your `kubectl` client to the cluster you want to test ExternalDNS with.

Begin by creating a Kubernetes secret to securely store your CloudFlare API key. This key will enable ExternalDNS to authenticate with CloudFlare:

```shell
kubectl create secret generic cloudflare-api-key --from-literal=apiKey=YOUR_API_KEY --from-literal=email=YOUR_CLOUDFLARE_EMAIL
```

And for API Token it should look like :

```shell
kubectl create secret generic cloudflare-api-key --from-literal=apiKey=YOUR_API_TOKEN
```

Ensure to replace YOUR_API_KEY with your actual CloudFlare API key and YOUR_CLOUDFLARE_EMAIL with the email associated with your CloudFlare account.

Then apply one of the following manifests file to deploy ExternalDNS.

### Using Helm

Create a values.yaml file to configure ExternalDNS to use CloudFlare as the DNS provider. This file should include the necessary environment variables:

```yaml
provider:
  name: cloudflare
env:
  - name: CF_API_KEY
    valueFrom:
      secretKeyRef:
        name: cloudflare-api-key
        key: apiKey
  - name: CF_API_EMAIL
    valueFrom:
      secretKeyRef:
        name: cloudflare-api-key
        key: email
```

Use this in your values.yaml, if you are using API Token:

```yaml
provider:
  name: cloudflare
env:
  - name: CF_API_TOKEN
    valueFrom:
      secretKeyRef:
        name: cloudflare-api-key
        key: apiKey
```

Finally, install the ExternalDNS chart with Helm using the configuration specified in your values.yaml file:

```shell
helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/
```

```shell
helm repo update
```

```shell
helm upgrade --install external-dns external-dns/external-dns --values values.yaml
```

### Manifest (for clusters without RBAC enabled)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: external-dns
spec:
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: external-dns
  template:
    metadata:
      labels:
        app: external-dns
    spec:
      containers:
        - name: external-dns
          image: registry.k8s.io/external-dns/external-dns:v0.18.0
          args:
            - --source=service # ingress is also possible
            - --domain-filter=example.com # (optional) limit to only example.com domains; change to match the zone created above.
            - --zone-id-filter=023e105f4ecef8ad9ca31a8372d0c353 # (optional) limit to a specific zone.
            - --provider=cloudflare
            - --cloudflare-proxied # (optional) enable the proxy feature of Cloudflare (DDOS protection, CDN...)
            - --cloudflare-dns-records-per-page=5000 # (optional) configure how many DNS records to fetch per request
            - --cloudflare-regional-services # (optional) enable the regional hostname feature that configure which region can decrypt HTTPS requests
            - --cloudflare-region-key="eu" # (optional) configure which region can decrypt HTTPS requests
            - --cloudflare-record-comment="provisioned by external-dns" # (optional) configure comments for provisioned records; <=100 chars for free zones; <=500 chars for paid zones
         env:
            - name: CF_API_KEY
              valueFrom:
                secretKeyRef:
                  name: cloudflare-api-key
                  key: apiKey
            - name: CF_API_EMAIL
              valueFrom:
                secretKeyRef:
                  name: cloudflare-api-key
                  key: email
```

### Manifest (for clusters with RBAC enabled)

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: external-dns
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: external-dns
rules:
  - apiGroups: [""]
    resources: ["services","pods"]
    verbs: ["get","watch","list"]
  - apiGroups: ["discovery.k8s.io"]
    resources: ["endpointslices"]
    verbs: ["get","watch","list"]
  - apiGroups: ["extensions","networking.k8s.io"]
    resources: ["ingresses"]
    verbs: ["get","watch","list"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: external-dns-viewer
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: external-dns
subjects:
- kind: ServiceAccount
  name: external-dns
  namespace: default
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: external-dns
spec:
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: external-dns
  template:
    metadata:
      labels:
        app: external-dns
    spec:
      serviceAccountName: external-dns
      containers:
        - name: external-dns
          image: registry.k8s.io/external-dns/external-dns:v0.18.0
          args:
            - --source=service # ingress is also possible
            - --domain-filter=example.com # (optional) limit to only example.com domains; change to match the zone created above.
            - --zone-id-filter=023e105f4ecef8ad9ca31a8372d0c353 # (optional) limit to a specific zone.
            - --provider=cloudflare
            - --cloudflare-proxied # (optional) enable the proxy feature of Cloudflare (DDOS protection, CDN...)
            - --cloudflare-dns-records-per-page=5000 # (optional) configure how many DNS records to fetch per request
            - --cloudflare-regional-services # (optional) enable the regional hostname feature that configure which region can decrypt HTTPS requests
            - --cloudflare-region-key="eu" # (optional) configure which region can decrypt HTTPS requests
            - --cloudflare-record-comment="provisioned by external-dns" # (optional) configure comments for provisioned records; <=100 chars for free zones; <=500 chars for paid zones
          env:
            - name: CF_API_KEY
              valueFrom:
                secretKeyRef:
                  name: cloudflare-api-key
                  key: apiKey
            - name: CF_API_EMAIL
              valueFrom:
                secretKeyRef:
                  name: cloudflare-api-key
                  key: email
```

## Deploying an Nginx Service

Create a service file called 'nginx.yaml' with the following contents:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - image: nginx
        name: nginx
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
  annotations:
    external-dns.alpha.kubernetes.io/hostname: example.com
    external-dns.alpha.kubernetes.io/ttl: "120" #optional
spec:
  selector:
    app: nginx
  type: LoadBalancer
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
```

Note the annotation on the service; use the same hostname as the Cloudflare DNS zone created above. The annotation may also be a subdomain
of the DNS zone (e.g. 'www.example.com').

By setting the TTL annotation on the service, you have to pass a valid TTL, which must be 120 or above.
This annotation is optional, if you won't set it, it will be 1 (automatic) which is 300.
For Cloudflare proxied entries, set the TTL annotation to 1 (automatic), or do not set it.

ExternalDNS uses this annotation to determine what services should be registered with DNS.  Removing the annotation
will cause ExternalDNS to remove the corresponding DNS records.

Create the deployment and service:

```shell
kubectl create -f nginx.yaml
```

Depending where you run your service it can take a little while for your cloud provider to create an external IP for the service.

Once the service has an external IP assigned, ExternalDNS will notice the new service IP address and synchronize
the Cloudflare DNS records.

## Verifying Cloudflare DNS records

Check your [Cloudflare dashboard](https://www.cloudflare.com/a/dns/example.com) to view the records for your Cloudflare DNS zone.

Substitute the zone for the one created above if a different domain was used.

This should show the external IP address of the service as the A record for your domain.

## Cleanup

Now that we have verified that ExternalDNS will automatically manage Cloudflare DNS records, we can delete the tutorial's example:

```shell
kubectl delete -f nginx.yaml
kubectl delete -f externaldns.yaml
```

## Setting cloudflare-proxied on a per-ingress basis

Using the `external-dns.alpha.kubernetes.io/cloudflare-proxied: "true"` annotation on your ingress, you can specify if the proxy feature of Cloudflare should be enabled for that record. This setting will override the global `--cloudflare-proxied` setting.

## Setting cloudlfare regional services

With Cloudflare regional services you can restrict which data centers can decrypt and serve HTTPS traffic.

Configuration of Cloudflare Regional Services is enabled by the `--cloudflare-regional-services` flag.
A default region can be defined using the `--cloudflare-region-key` flag.

Using the `external-dns.alpha.kubernetes.io/cloudflare-region-key` annotation on your ingress, you can specify the region for that record.

An empty string will result in no regional hostname configured.

**Accepted values for region key include:**

- `eu`: European Union data centers only
- `us`: United States data centers only
- `ap`: Asia-Pacific data centers only
- `fedramp`: US public sector (FedRAMP) data centers
- `in`: India data centers only
- `ca`: Canada data centers only
- `jp`: Japan data centers only
- `kr`: South Korea data centers only
- `br`: Brazil data centers only
- `za`: South Africa data centers only
- `ae`: United Arab Emirates data centers only

For the most up-to-date list and details, see the [Cloudflare Regional Services documentation](https://developers.cloudflare.com/data-localization/regional-services/get-started/).

Currently, requires SuperAdmin or Admin role.

## Setting cloudflare-custom-hostname

Automatic configuration of Cloudflare custom hostnames (using A/CNAME DNS records as custom origin servers) is enabled by the `--cloudflare-custom-hostnames` flag and the `external-dns.alpha.kubernetes.io/cloudflare-custom-hostname: <custom hostname>` annotation.

Multiple hostnames are supported via a comma-separated list: `external-dns.alpha.kubernetes.io/cloudflare-custom-hostname: <custom hostname 1>,<custom hostname 2>`.

See [Cloudflare for Platforms](https://developers.cloudflare.com/cloudflare-for-platforms/cloudflare-for-saas/domain-support/) for more information on custom hostnames.

This feature is disabled by default and supports the `--cloudflare-custom-hostnames-min-tls-version` and `--cloudflare-custom-hostnames-certificate-authority` flags.

`--cloudflare-custom-hostnames-certificate-authority` defaults to `none`, which explicitly means no Certificate Authority (CA) is set when using the Cloudflare API. Specifying a custom CA is only possible for enterprise accounts.

The custom hostname DNS must resolve to the Cloudflare DNS record (`external-dns.alpha.kubernetes.io/hostname`) for automatic certificate validation via the HTTP method. It's important to note that the TXT method does not allow automatic validation and is not supported.

Requires [Cloudflare for SaaS](https://developers.cloudflare.com/cloudflare-for-platforms/cloudflare-for-saas/) product and "SSL and Certificates" API permission.

**Note:** Due to using the legacy cloudflare-go v0 API for custom hostname management, the custom hostname page size is fixed at 50. This limitation will be addressed in a future migration to the v4 SDK.

## Using CRD source to manage DNS records in Cloudflare

Please refer to the [CRD source documentation](../sources/crd.md#example) for more information.

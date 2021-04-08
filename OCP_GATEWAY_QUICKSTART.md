The following guide provides instructions for using [Gateway API][1] on OpenShift.

Before proceeding, install OpenShift and ensure your oc client is configured to access the cluster. The OpenShift
version tested is `4.8.0-0.nightly-2021-04-08-005413`.

Install Contour Operator:
```shell
$ oc apply -f https://raw.githubusercontent.com/projectcontour/contour-operator/v1.14.0/examples/operator/operator.yaml
```

Verify the availability of the operator:
```shell
$ oc get deployment/contour-operator -n contour-operator
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
contour-operator   1/1     1            1           12m
```

Add `contour` and `contour-certgen` service accounts to the "nonroot" scc. See [issue 112][2] for additional background
on this requirement. __Note:__ The example below uses `projectcontour` as the namespace of the service accounts. This
namespace should match the Gateway's namespace.
```shell
$ oc adm policy add-scc-to-user nonroot system:serviceaccount:projectcontour:contour
$ oc adm policy add-scc-to-user nonroot system:serviceaccount:projectcontour:contour-certgen
```

Install the [Gateway][3] and dependent resources:
```shell
$ oc apply -f https://raw.githubusercontent.com/projectcontour/contour-operator/v1.14.0/examples/gateway/gateway.yaml
```
__Note:__ Envoy will be exposed using a Kubernetes LoadBalancer service. Replace `gateway.yaml` with
`gateway-nodeport.yaml` to use a NodePort service instead.

Verify the status of the [Contour][4] resource. This resource is referenced by a GatewayClass and contains
implementation-specific configuration for managing infrastructure through Gateway API. For example, using a
load-balancer for the Envoy pods:
```shell
$ oc get contour/contour-gateway-sample -n contour-operator
NAME                     READY   REASON
contour-gateway-sample   True    GatewayClassAdmitted
```
A `Ready=True` status indicates the Contour is valid and references a GatewayClass that is "Admitted".

Verify the status of the Gateway API resources, first the [GatewayClass][5]:
```shell
$ oc get gc/sample-gatewayclass -o yaml
apiVersion: networking.x-k8s.io/v1alpha1
kind: GatewayClass
  name: sample-gatewayclass
...
spec:
  controller: projectcontour.io/contour-operator
  parametersRef:
    group: operator.projectcontour.io
    kind: Contour
    name: contour-gateway-sample
    namespace: contour-operator
    scope: Namespace
status:
  conditions:
  - lastTransitionTime: "2021-04-08T17:16:34Z"
    message: Owned by Contour Operator.
    reason: Owned
    status: "True"
    type: Admitted
```
The `projectcontour.io/contour-operator` value of the GatewayClass indicates Contour Operator is managing Gateways of
this class. The GatewayClass uses the `parametersRef` field to reference the Contour resource from the previous step. 

Next, verify the status of the `Gateway` resource:
```shell
$ oc get gateway/contour -n projectcontour -o yaml
apiVersion: networking.x-k8s.io/v1alpha1
kind: Gateway
  name: contour
  namespace: projectcontour
...
spec:
  gatewayClassName: sample-gatewayclass
  listeners:
  - port: 80
    protocol: HTTP
    routes:
      group: networking.x-k8s.io
      kind: HTTPRoute
      namespaces:
        from: Same
        selector: {}
      selector:
        matchLabels:
          app: kuard
  - port: 443
    protocol: HTTPS
    routes:
      group: networking.x-k8s.io
      kind: HTTPRoute
      namespaces:
        from: Same
        selector: {}
      selector:
        matchLabels:
          app: kuard
status:
  conditions:
  - lastTransitionTime: "2021-04-08T17:16:37Z"
    message: The Gateway is ready to serve routes.
    reason: GatewayReady
    status: "True"
    type: Ready
```
The Gateway references the GatewayClass named "sample-gatewayclass" from the previous step and contains two listeners,
one for HTTP and one for HTTPS. The listeners will bind to routes that meet the following criteria:
- The route is of type HTTPRoute.
- The routes contains label `app: kuard`.
- The route resides in the same namespace as the Gateway.

The "Ready=True" status condition indicates the infrastructure is ready to begin
serving routes.

__Notes:__
- Although both listeners are configured, Contour v1.14.0 only supports the HTTP listener.
- It may take a few minutes for the Gateway to become available while the infrastructure is being provisioned.

(Optional) Verify that all pods created by the Gateway are running:
```shell
$ oc get po -n projectcontour
NAME                       READY   STATUS    RESTARTS   AGE
contour-6b55fbd747-8jkx2   1/1     Running   0          52m
contour-6b55fbd747-9mtx6   1/1     Running   0          52m
envoy-m5xkd                2/2     Running   0          52m
envoy-pt8k2                2/2     Running   0          52m
envoy-s4xr8                2/2     Running   0          52m
```
__Note:__ The number of Envoy pods will depend on the amount of worker nodes in your cluster.

Run [kuard][6] as a test workload:
```shell
$ oc apply -f https://raw.githubusercontent.com/projectcontour/contour-operator/v1.14.0/examples/gateway/kuard/kuard.yaml
```

Verify the status of the test workload:
```shell
$ oc get po,svc,httproute -n projectcontour -l app=kuard
NAME                         READY   STATUS    RESTARTS   AGE
pod/kuard-798585497b-9mvwh   1/1     Running   0          5s
pod/kuard-798585497b-kcjnn   1/1     Running   0          5s
pod/kuard-798585497b-lnhsn   1/1     Running   0          5s

NAME            TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)   AGE
service/kuard   ClusterIP   10.96.157.48   <none>        80/TCP    5s

NAME                                  HOSTNAMES
httproute.networking.x-k8s.io/kuard   ["local.projectcontour.io"]
```
All kuard pods should be running, and the kuard service should exist. The kuard [HTTPRoute][7] will route HTTP requests
for "local.projectcontour.io" to kuard pods through the Envoy proxies.

Review the details of the HTTPRoute:
```shell
$ oc get httproute/kuard -n projectcontour -o yaml
apiVersion: networking.x-k8s.io/v1alpha1
kind: HTTPRoute
metadata:
  labels:
    app: kuard
  name: kuard
  namespace: projectcontour
...
spec:
  gateways:
    allow: SameNamespace
  hostnames:
  - local.projectcontour.io
  rules:
  - forwardTo:
    - port: 80
      serviceName: kuard
      weight: 1
    matches:
    - path:
        type: Prefix
        value: /
status:
  gateways:
  - conditions:
    - lastTransitionTime: "2021-04-08T18:13:22Z"
      message: Valid HTTPRoute
      observedGeneration: 1
      reason: Valid
      status: "True"
      type: Admitted
    gatewayRef:
      name: contour
      namespace: projectcontour
```
The HTTPRoute should surface the "Admitted=True" status condition and indicate the Gateways that have admitted the
route.

The route binds to Gateways in the same namespace, baring a Gateway exists and contains:
```shell
...
routes:
  group: networking.x-k8s.io
  kind: HTTPRoute
  namespaces:
    from: Same
```
The route will match all HTTP requests with host header "local.projectcontour.io" and process the `rules` to perform a
routing decision.

__Note__: Since the Gateway in the previous step uses selectors for route binding, the `app: kuard` label in the
HTTPRoute is required.

Test by curl'ing the route hostname. Since a DNS record does not exist for this hostname, the `-H` flag is used with
curl to set the host header:
```shell
$ export GATEWAY=$(oc -n projectcontour get svc/envoy -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
$ curl -H "Host: local.projectcontour.io" -s -o /dev/null -w "%{http_code}" "http://$GATEWAY/"
```
__Note:__ Replace `hostname` in the json path with `ip` if your cloud provider uses IP addresses instead of hostnames
for load-balancer services.

A 200 HTTP status code should be returned.

Verify the curl request was serviced by Envoy:
```shell
$ oc logs ds/envoy -c envoy -n projectcontour | grep curl
Found 3 pods, using pod/envoy-s4xr8
[2021-04-08T18:17:14.472Z] "GET / HTTP/1.1" 200 - 0 1700 2 2 "10.0.88.93" "curl/7.64.1" "1d5453d0-ab6f-42fe-9c3b-a03d79dd363a" "local.projectcontour.io" "10.129.2.13:8080"
```
__Note__: The example above defaulted to pod `envoy-s4xr8t` since the DaemonSet has 3 running pods. Use a different pod
if the curl request does not appear in the logs.

Verify the curl request was received by one of the kuard pods:
```shell
$ oc logs deploy/kuard -n projectcontour
Found 3 pods, using pod/kuard-798585497b-vggjb
2021/04/08 18:13:26 Starting kuard version: v0.8.1-1
...
2021/04/08 18:17:14 10.129.2.12:48414 GET /
```
__Note__: The example above defaulted to pod `pod/kuard-798585497b-vggjb` since the Deployment has 3 running pods. Use a
different pod if the curl request does not appear in the logs.

Refer to Contour's [Gateway API user guide][7] for additional details.

[1]: https://gateway-api.sigs.k8s.io/
[2]: https://github.com/projectcontour/contour-operator/issues/112
[3]: https://gateway-api.sigs.k8s.io/gateway/
[4]: https://github.com/projectcontour/contour-operator/blob/release-1.14/api/v1alpha1/contour_types.go
[5]: https://gateway-api.sigs.k8s.io/gatewayclass/
[6]: https://github.com/kubernetes-up-and-running/kuard
[7]: https://gateway-api.sigs.k8s.io/httproute/
[8]: https://projectcontour.io/guides/gateway-api/

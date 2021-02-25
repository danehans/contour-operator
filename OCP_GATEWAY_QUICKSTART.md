The following guide provides instructions for using [Gateway API][1] with Contour on OpenShift.

Before proceeding, install OpenShift and ensure kubectl is configured to access the cluster. The
OpenShift version tested is `4.7.0-0.nightly-2021-02-25-102400`.

Install Contour Operator:
```shell
$ kubectl apply -f https://raw.githubusercontent.com/projectcontour/contour-operator/v1.13.0/examples/operator/operator.yaml
```

Verify the availability of the operator:
```shell
$ kubectl get deployment/contour-operator -n contour-operator
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
contour-operator   1/1     1            1           12m
```

Add `contour` and `contour-certgen` service accounts to the nonroot scc:
```shell
$ oc adm policy add-scc-to-user nonroot system:serviceaccount:projectcontour:contour
$ oc adm policy add-scc-to-user nonroot system:serviceaccount:projectcontour:contour-certgen
```
See [issue 112][2] for background on the scc requirement.

Install the [Gateway][3] and dependent resources:
```shell
$ kubectl apply -f https://raw.githubusercontent.com/projectcontour/contour-operator/v1.13.0/examples/gateway/gateway.yaml
```

__Note:__ Envoy will be exposed using a LoadBalancer service. Replace `gateway.yaml` with `gateway-nodeport.yaml` to use
a NodePort service instead.

Verify that all pods in the projectcontour namespace are running:
```shell
$ kubectl get po -n projectcontour
NAME                         READY   STATUS      RESTARTS   AGE
contour-768547cfb8-c2rhn     1/1     Running     0          2m
contour-768547cfb8-q866f     1/1     Running     0          2m
contour-certgen-main-rb2h2   0/1     Completed   0          92s
envoy-d5djm                  2/2     Running     0          2m41s
envoy-gjwz5                  2/2     Running     0          2m41s
envoy-hbg6j                  2/2     Running     0          2m41s
```

The number of Envoy pods will depend on how many worker nodes are in your cluster.

__Note__: It may take a few minutes for the Contour to become available.

Run a test workload:
```shell
$ kubectl apply -f https://raw.githubusercontent.com/projectcontour/contour-operator/v1.13.0/examples/gateway/kuard/kuard.yaml
```

Verify the status of the test workload:
```shell
$ kubectl get po,svc,httproute -n projectcontour -l app=kuard
NAME                         READY   STATUS    RESTARTS   AGE
pod/kuard-798585497b-9mvwh   1/1     Running   0          5s
pod/kuard-798585497b-kcjnn   1/1     Running   0          5s
pod/kuard-798585497b-lnhsn   1/1     Running   0          5s

NAME            TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)   AGE
service/kuard   ClusterIP   10.96.157.48   <none>        80/TCP    5s

NAME                                  HOSTNAMES
httproute.networking.x-k8s.io/kuard   ["local.projectcontour.io"]
```

Note that the application is exposed using an [HTTPRoute][4] that will route all HTTP requests for
"local.projectcontour.io" to service kuard.

Test by curl'ing the application hostname:
```shell
$ export GATEWAY=$(kubectl -n projectcontour get svc/envoy -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
$ curl -H "Host: local.projectcontour.io" -s -o /dev/null -w "%{http_code}" "http://$GATEWAY/"
```

A 200 HTTP status code should be returned.

Verify the curl request was serviced by Envoy:
```shell
$ kubectl logs ds/envoy -c envoy -n projectcontour | grep curl
Found 3 pods, using pod/envoy-g86st
[2021-02-03T17:17:24.009Z] "GET / HTTP/1.1" 200 - 0 1748 1 1 "10.0.79.141" "curl/7.64.1" "2c53c9ba-46a2-4527-8b41-03ea9041bd2d" "a811b15855e1f428d8a834d0a86c3668-573506534.us-east-2.elb.amazonaws.com" "10.129.2.13:8080"
```
__Note__: The example above defaulted to pod `envoy-g86st` since the daemonset has 3 running pods. Use a different pod
if the curl request does not appear in the logs.

Refer to Contour's [Gateway API user guide][5] for additional details.

[1]: https://gateway-api.sigs.k8s.io/
[2]: https://github.com/projectcontour/contour-operator/issues/112
[3]: https://gateway-api.sigs.k8s.io/gateway/
[4]: https://gateway-api.sigs.k8s.io/httproute/
[5]: https://projectcontour.io/guides/gateway-api/

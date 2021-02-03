The following guide provides instructions for running Contour Operator on OpenShift.

Install OpenShift. The OpenShift version tested is `4.7.0-0.nightly-2021-02-01-180932`.

Install Contour Operator:
```shell
$ kubectl apply -f https://raw.githubusercontent.com/projectcontour/contour-operator/v1.12.0/examples/operator/operator.yaml
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
See [issue 112][1] for background on the scc requirement.

Create an instance of the `Contour` custom resource:
```shell
$ cat <<EOF | kubectl apply -f -
apiVersion: operator.projectcontour.io/v1alpha1
kind: Contour
metadata:
  name: contour-sample
spec: {}
EOF
```

Verify the Contour is available.
```shell
$ kubectl get contour/contour-sample
NAME             READY   REASON
contour-sample   True    ContourAvailable
```
__Note__: It may take a few minutes for the Contour to become available:

Test ingress by running a workload:
```shell
$ kubectl apply -f https://projectcontour.io/examples/kuard.yaml
```

Verify the status of the test workload:
```shell
$ kubectl get po,svc,ing -l app=kuard
NAME                         READY   STATUS    RESTARTS   AGE
pod/kuard-798585497b-7kvdp   1/1     Running   0          16s
pod/kuard-798585497b-c74z8   1/1     Running   0          16s
pod/kuard-798585497b-tqsq8   1/1     Running   0          16s

NAME            TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
service/kuard   ClusterIP   172.30.96.209   <none>        80/TCP    16s

NAME                              CLASS    HOSTS   ADDRESS                                                                  PORTS   AGE
ingress.networking.k8s.io/kuard   <none>   *       a811b15855e1f428d8a834d0a86c3668-573506534.us-east-2.elb.amazonaws.com   80      16s
```

Test ingress by curl'ing the ingress address:
```shell
$ curl -s -o /dev/null -w "%{http_code}" http://a811b15855e1f428d8a834d0a86c3668-573506534.us-east-2.elb.amazonaws.com
200
```

Verify the curl request was serviced by Envoy:
```shell
$ kubectl logs ds/envoy -c envoy -n projectcontour | grep curl
Found 3 pods, using pod/envoy-g86st
[2021-02-03T17:17:24.009Z] "GET / HTTP/1.1" 200 - 0 1748 1 1 "10.0.79.141" "curl/7.64.1" "2c53c9ba-46a2-4527-8b41-03ea9041bd2d" "a811b15855e1f428d8a834d0a86c3668-573506534.us-east-2.elb.amazonaws.com" "10.129.2.13:8080"
```
__Note__: The example above defaulted to pod `envoy-g86st` since the daemonset has 3 running pods. Use a different pod
if the curl request does not appear in the logs.

[1]: https://github.com/projectcontour/contour-operator/issues/112

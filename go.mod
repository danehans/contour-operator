module github.com/projectcontour/contour-operator

go 1.15

require (
	github.com/docker/distribution v2.7.1+incompatible
	github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr v0.2.0 //indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/projectcontour/contour v1.9.0
	github.com/sclevine/agouti v3.0.0+incompatible // indirect
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/controller-runtime v0.8.2
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/gateway-api v0.2.0
)

// TODO [danehans]: Remove when Envoy API is merged.
replace github.com/projectcontour/contour => github.com/danehans/contour v0.13.0-beta.2.0.20210413191629-a9ce4434d46e

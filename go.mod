module github.com/LimKianAn/syncrd

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.15.1
	github.com/onsi/gomega v1.10.4
	k8s.io/api v0.20.5
	k8s.io/apiextensions-apiserver v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/client-go v0.20.5
	repo-url v0.0.0-00010101000000-000000000000
	sigs.k8s.io/controller-runtime v0.8.3
)

replace repo-url => github.com/metal-stack/firewall-controller v1.0.8

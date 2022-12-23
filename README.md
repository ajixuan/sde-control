Learning the kubernetes operator

# Prerequisites
go version 1.19+

# Docs:
Tutorial:
https://book.kubebuilder.io/cronjob-tutorial/gvks.html

### Resources:
https://opensource.com/article/22/9/packaging-job-scripts-kubernetes-operators
https://betterprogramming.pub/build-a-kubernetes-operator-in-10-minutes-11eec1492d30
https://cloudark.medium.com/kubernetes-operators-and-helm-it-takes-two-to-tango-3ff6dcf65619#:~:text=Helm%20and%20Operators%20are%20complementary,managing%20application%20workloads%20on%20Kubernetes.

### Arguments for Operator:
https://developer.ibm.com/learningpaths/why-when-kubernetes-operators/summary/

### Example of Cloud Native PG:
https://github.com/cloudnative-pg/cloudnative-pg/blob/main/docs/src/bootstrap.md

# Operations
### Steps to Initialize a Kubebuilder project (incomplete):
```
kubebuilder init --domain sde.domain --repo sde.domain/sdeController
kubebuilder create api --group sde-controller --version v1beta1 --kind SdeController
```

### Steps to deploy demo:
1. Make sure your Kubeconfig is set to the right context and cluster
1. Change the `namespace` attribute of `./config/default/kustomization.yaml` file to match your namespace
1. Build and install the controller Custom Resource to your cluster:
```
make build
make install
```
1. Build and Deploy the controller container:
```
export IMG=docker-dev.sdelements.com/dev_sde/sde-controller:0.0.1
make docker-build
make docker-push
make docker-deploy
```
1. Deploy an instance of the SDE Custom resource:
```
kubectl apply -f config/samples/ --namespace <your namespace>
```
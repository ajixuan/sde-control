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
Steps to Initialize:
```
kubebuilder init --domain sde.domain --repo sde.domain/sdeController
kubebuilder create api --group sde-controller --version v1beta1 --kind SdeController
```

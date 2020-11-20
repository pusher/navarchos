# Návarchos

> NOTE: this repository is currently **UNMAINTAINED** and is looking for new owner(s).
> See [#53](/../../issues/53) for more information.

Návarchos is a node replacing controller. It is intended to automate the upgrade
process of a Kubernetes cluster running immutable infrastructure. It provides a
declarative way to replace groups of nodes without supervision.

## Table of Contents

- [Návarchos](#n%c3%a1varchos)
  - [Table of Contents](#table-of-contents)
  - [Introduction](#introduction)
  - [Installation](#installation)
    - [Deploying to Kubernetes](#deploying-to-kubernetes)
      - [RBAC](#rbac)
    - [Configuration](#configuration)
      - [Leader Election](#leader-election)
      - [Sync period](#sync-period)
  - [Project Concepts](#project-concepts)
  - [Quick Start](#quick-start)
  - [Communication](#communication)
  - [Contributing](#contributing)
  - [License](#license)

## Introduction

Running Kubernetes on immutable infrastructure has [many
advantages](https://www.digitalocean.com/community/tutorials/what-is-immutable-infrastructure).
It introduces more consistency, reliability and provides a more robust
deployment process. It also mitigates or entirely prevents issues that are
common in mutable infrastructures, like configuration drift and snowflake
servers. However, there is one large disadvantage: an operator must manually
drain, cordon and bring up new nodes with modified configuration. Making changes
to large groups of nodes is tedious, error prone and can be extremely boring.
An example situation where one may encounter these problems could be when
upgrading to a new version of Kubernetes.

Návarchos aims to solve this problem. It is a node replacing controller. It
automates this process so you (the operator) do not have to oversee it.

Návarchos has two central concepts: `NodeRollout`s and `NodeReplacement`s.  A
`NodeRollout` provides a way to select nodes or groups of nodes for replacement, and to
set the order in which they are replaced. A `NodeRollout` object tracks the
overall progress of the rollout procedure.

A `NodeReplacement` replaces a node.  It owns the replacement options and
process.  The process currently consists of cordoning and draining a node. A
`NodeReplacement` is created for each node defined by a `NodeRollout`. Each
`NodeReplacement` tracks the progress of the node that owns it.

This approach was taken to try to minimise the amount of context switching
required to replace a group of nodes. Generally as a Kubernetes operator you may
want to work at the level of the Kubernetes API, but when modifying the
underlying infrastructure this becomes difficult: one must interact with custom
cloud provider tools and move between the various layers of the infrastructure
repeatedly. **This is the pain point Návarchos aims to solve. What if you could
replace nodes only using Kubernetes principles, without
supervision?**

## Installation

### Deploying to Kubernetes

Návarchos is a [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
based project.

A public docker image is available on
[Quay](https://quay.io/repository/pusher/navarchos).

```text
quay.io/pusher/navarchos
```

Firstly, install the CRDs:

```bash
$ kubectl create -f config/crds
```

After this completes the api resources should be available:

```bash
$ kubectl api-resources --api-group=navarchos.pusher.com
nodereplacements                               navarchos.pusher.com           false        NodeReplacement
noderollouts                                   navarchos.pusher.com           false        NodeRollout
```

Then deploy the controllers:

```bash
$ kubectl create -f config/deploy
```

You should be able to observe the controllers starting:

```bash
$ kubectl logs -n kube-system -l app=navarchos --follow
kube-system/navarchos-5ff95cd5bc-z2k2k[controller]: I1021 13:11:42.012584       1 main.go:58] entrypoint "level"=0 "msg"="setting up client for manager"
kube-system/navarchos-5ff95cd5bc-z2k2k[controller]: I1021 13:11:42.013202       1 main.go:66] entrypoint "level"=0 "msg"="setting up manager"
kube-system/navarchos-5ff95cd5bc-9hj62[controller]: I1021 13:11:42.098501       1 main.go:58] entrypoint "level"=0 "msg"="setting up client for manager"
kube-system/navarchos-5ff95cd5bc-9hj62[controller]: I1021 13:11:42.108141       1 main.go:66] entrypoint "level"=0 "msg"="setting up manager"
kube-system/navarchos-5ff95cd5bc-z2k2k[controller]: I1021 13:11:42.962981       1 listener.go:40] controller-runtime/metrics "level"=0 "msg"="metrics server is starting to listen"
"addr"=":8080"
```

Now you're ready to jump to the [quick start](#quick-start).

#### RBAC

If you are using
[RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) within
your cluster you must grant the service account used by your Návarchos instance
permission cordon nodes, evict pods and emit events.

The deployment in `config/deploy` assumes that you are using RBAC and has
appropriate `ClusterRole`s and `ClusterRoleBinding`s.

### Configuration

The following section details the various configuration options that Návarchos
provides at the controller level.

#### Leader Election

Návarchos can be run in an active-standby HA configuration using Kubernetes
leader election. When leader election is enabled each Pod will attempt to
become leader and whichever is successful will become the active or leader
controller. The leader will perform all of the reconciliation of operations.

The standby Pods will sit and wait until the leader goes away and then one
standby will be promoted to leader.

Leader election is enabled by setting the following flags:

```yaml
--leader-election=true
--leader-election-id=<name-of-leader-election-configmap>
--leader-election-namespace=<namespace-controller-runs-in>
```

The deployment in `config/deploy` enables leader election by default.

#### Sync period

The controller uses Kubernetes informers to cache resources and reduce load on
the Kubernetes API server. Every informer has a sync period, after which it will
refresh all resources in its cache. At this point every item in the cache
is queued for reconciliation by the controller.

Therefore by setting the following flag;

```yaml
--sync-period=5m // Default value of 5m (5 minutes)
```

You can ensure that every resource will be reconciled at least every 5 minutes.

## Project Concepts

A `NodeRollout` provides a way to select a node or groups of nodes for
replacement, and to set priorities for those replacements. A `NodeReplacement`
is created for each node specified by a `NodeRollout`. A node rollout may look
something like this:

```yaml
apiVersion: navarchos.pusher.com/v1alpha1
kind: NodeRollout
metadata:
 # A NodeReplacement will be created with a name 'rollout-<random-string>'
  generateName: "rollout-"
spec:
  nodeSelectors:
    - replacement:
        priority: 20
      matchLabels:
        "kubernetes.io/role": "master"
    - replacement:
        priority: 10
      matchLabels:
        "kubernetes.io/role": "worker"
```

The priority set for the replacement ensures that Návarchos will not replace any
`worker`s until all other replacements specified at a higher priority have
completed.

Because the `generateName` field is set this example can't be created by calling
`kubectl apply`. It must be created using `kubectl create`. The `generateName`
field is used as it provides unique name with a common prefix. This avoids name
clashes and allows you to redo a replacement easily when you have made further
changes to your configuraiton.


A `NodeRollout` can specify groups of nodes either using a label selector or by
specifying the nodes by name, using `matchName`:

```yaml
apiVersion: navarchos.pusher.com/v1alpha1
kind: NodeRollout
metadata:
  generateName: "rollout-"
spec:
  nodeNames:
    - replacement:
        priority: 20
      name: "node-1"
```

If multiple selectors match some of the same nodes within the same
`NodeRollout`, the selector with the highest priority will take precedence.

If two separate `NodeRollout`s match any node two `NodeReplacement`s will be
created. The one with the highest priority will be processed first.

There is no order guarantee for nodes with the same priority.

For a comprehensive example see [rollout.yml](rollout.yml)

## Quick Start

If you haven't yet got Návarchos running on your cluster see
[Installation](#installation) for details on how to get it running.

Modify the example `NodeRollout` found in the [project
concepts](#project-concepts) section. You may want to change the label selector.
Multiple replacements with different priorities can be defined in one rollout.

Next, create the rollout:

```bash
$ kubectl create -f nodeRollout.yaml
noderollout.navarchos.pusher.com/rollout-6lwkc created
```

The remainder of the process should be automatic. The `NodeRollout` controller
creates `NodeReplacement`s for each node specified, the `NodeReplacement`s
take care of replacing nodes.

The progress can be observed through the logs or through the status of the
`NodeRollout` and `NodeReplacement` objects. Both have completion
timestamps.

## Communication

- Found a bug? Please open an issue.
- Have a feature request. Please open an issue.
- If you want to contribute, please submit a pull request.

## Contributing

Please see our [Contributing](CONTRIBUTING.md) guidelines.

## License

This project is licensed under Apache 2.0 and a copy of the license is available
[here](LICENSE).

Kubernetes Scheduler: RateLimit
===============================

A custom scheduler for rate limiting large busts of Pod creations.

## Use cases

* Avoid stampedes on shared backend infrastructure eg. Shared database cluster with 100s for backup tasks.

## How this is implemented

This scheduler is a custom [permit](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/#permit) plugin built ontop of the Kubernetes Scheduler Framework.

This approach gives us the benefit of a custom scheduler as well as all the existing scheduler logic to fallback on when our business logic is executed.

## Installation

```bash
kubectl apply -f deploy/
```

## Usage

See the [example](example/pods.yaml) for Pod configuration.

## Resources

This is a great example repository to draw inspiration from:

https://github.com/kubernetes-sigs/scheduler-plugins

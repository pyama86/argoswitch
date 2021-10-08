# ArgoSwitch

ArgoSwitch is updating ArgoCD Application Config.

![top](./images/top.png)
![secondry](./images/secondry.png)

## usage

You can defined behavior when cluster state is change.
```
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: example
  annotations:
    argoswitch.github.io/primary: sync
    argoswitch.github.io/secondry: disable
    argoswitch.github.io/service-out: delete
```

- auto-sync: do sync and enable auto sync.
- sync: do sync.
- disable, disable-sync : stop auto sync.
- delete, delete-app: delete application.
- delete-resource: delete resources and stop auto sync.

## Author
@pyama86

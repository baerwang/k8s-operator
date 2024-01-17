## build operator

```shell
docker build -t baerwang/ingress-manage:1.0.0 .
```

## create manifests

```shell
mkdir manifests
```

## role

```shell
kubectl create clusterrole ingress-manager-crole --verb=list,watch,create,update,delete --resource=ingresses,service --dry-run=client -o yaml > manifests/ingress-manager-crole.yaml
```

## cluster role binding

```shell
kubectl create clusterrolebinding ingress-manager-crb --clusterrole=ingress-manager-crole --serviceaccount=default:default --dry-run=client -o yaml > manifests/ingress-manager-crb.yaml
```

## deployment

```shell
kubectl create deployment ingress-manager --image=baerwang/ingress-manage:1.0.0 --dry-run=client -o yaml > manifests/ingress-manager.yaml
```

### deployment image policy change use local images `Never`

```txt
- image: baerwang/ingress-manage:1.0.0
  name: ingress-manage
  imagePullPolicy: Never
  resources: {}
```

## apply

```shell
kubectl apply -f manifests
```

## run nginx

```shell
kubectl run nginx-demo --image=nginx:latest
```

## expose

```shell
kubectl expose pods/nginx-demo --port 8222 --target-port 80
```

## edit srv

```shell
kubectl edit service/nginx-demo
```

```txt
metadata:
  annotations:
    ingress/http: "true"
```
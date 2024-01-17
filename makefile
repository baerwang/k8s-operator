build:
	docker build -t baerwang/ingress-manage:1.0.0 .

dir:
	mkdir manifests

role:
	kubectl create clusterrole ingress-manager-crole --verb=list,watch,create,update,delete --resource=ingresses,service --dry-run=client -o yaml > manifests/ingress-manager-crole.yaml

rb:
	kubectl create clusterrolebinding ingress-manager-crb --clusterrole=ingress-manager-crole --serviceaccount=default:default --dry-run=client -o yaml > manifests/ingress-manager-crb.yaml

deployment:
	kubectl create deployment ingress-manager --image=baerwang/ingress-manage:1.0.0 --dry-run=client -o yaml > manifests/ingress-manager.yaml

apply:
	kubectl apply -f manifests

run: dir role rb deployment apply
	kubectl run nginx-demo --image=nginx:latest \
	kubectl expose pods/nginx-demo --port 8222 --target-port 80
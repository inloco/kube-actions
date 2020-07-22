docker: docker-build docker-tag docker-push

docker-build:
	docker build --build-arg DOCKER_VERSION=$(DOCKER_VERSION) --tag inloco/kube-actions:dind-$(DOCKER_VERSION) .

docker-tag:
	docker tag inloco/kube-actions:dind-$(DOCKER_VERSION) inloco/kube-actions:dind

docker-push:
	docker push inloco/kube-actions:dind-$(DOCKER_VERSION)
	docker push inloco/kube-actions:dind

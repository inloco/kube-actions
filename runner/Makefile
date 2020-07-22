docker: docker-build docker-tag docker-push

docker-build:
	docker build --build-arg RUNNER_VERSION=$(RUNNER_VERSION) --build-arg UBUNTU_VERSION=$(UBUNTU_VERSION) --tag inloco/kube-actions:runner-$(RUNNER_VERSION)-$(UBUNTU_VERSION) .

docker-tag:
	docker tag inloco/kube-actions:runner-$(RUNNER_VERSION)-$(UBUNTU_VERSION) inloco/kube-actions:runner-$(RUNNER_VERSION)
	docker tag inloco/kube-actions:runner-$(RUNNER_VERSION)-$(UBUNTU_VERSION) inloco/kube-actions:runner-$(UBUNTU_VERSION)
	docker tag inloco/kube-actions:runner-$(RUNNER_VERSION)-$(UBUNTU_VERSION) inloco/kube-actions:runner

docker-push:
	docker push inloco/kube-actions:runner-$(RUNNER_VERSION)-$(UBUNTU_VERSION)
	docker push inloco/kube-actions:runner-$(RUNNER_VERSION)
	docker push inloco/kube-actions:runner-$(UBUNTU_VERSION)
	docker push inloco/kube-actions:runner

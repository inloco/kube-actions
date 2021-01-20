SHELL = /bin/bash
.SHELLFLAGS = -o pipefail -ec

dind/% operator/% runner/%:
	$(MAKE) -C $(@D) $(@F)

continuous-upgrade:
	$(eval LATEST := $(shell curl -Lf https://api.github.com/repos/actions/runner/releases/latest | jq -r '.tag_name | ltrimstr("v")'))
	exit $(.SHELLSTATUS)
ifeq ($(shell sed -En 's/^RUNNER_VERSION \?= (.+)/\1/p' ./runner/Makefile), $(LATEST))
	echo 'Everything up-to-date'
else ifeq ($(shell git ls-remote --exit-code origin feat/actions-runner-v$(LATEST) >/dev/null)$(.SHELLSTATUS), 0)
	echo 'Pending merge'
else
	sed -Ei 's/^AGENT_VERSION \?= .*/AGENT_VERSION ?= $(LATEST)/' ./operator/Makefile
	sed -Ei 's/^RUNNER_VERSION \?= .*/RUNNER_VERSION ?= $(LATEST)/' ./runner/Makefile
	git add -A
	git commit -m 'feat: actions/runner#v$(LATEST)'
	git checkout -b 'feat/actions-runner-v$(LATEST)'
	git push -u "$$(git remote show)" "$$(git branch --show-current)"
	gh pr create -t 'Actions Runner v$(LATEST)' -b 'https://github.com/actions/runner/releases/tag/v$(LATEST)'
endif
.PHONY: continuous-upgrade

.ONESHELL:
.SHELLFLAGS = -o pipefail -ec

dind/% operator/% runner/%:
	$(MAKE) -C $(@D) $(@F)

continuous-upgrade:
	$(eval LATEST := $(shell curl -Lf https://api.github.com/repos/actions/runner/releases/latest | jq -r '.tag_name | ltrimstr("v")'))
	exit $(.SHELLSTATUS)
ifeq ($(shell sed -En 's/^RUNNER_VERSION \?= (.+)/\1/p' ./runner/Makefile), $(LATEST))
	echo 'Everything up-to-date'
else
	sed -Ei 's/^AGENT_VERSION \?= .*/AGENT_VERSION ?= $(LATEST)/' ./operator/Makefile
	sed -Ei 's/^RUNNER_VERSION \?= .*/RUNNER_VERSION ?= $(LATEST)/' ./runner/Makefile
	git add -A
	git commit -m 'feat: actions/runner#v$(LATEST)'
	git checkout -b 'feat/actions-runner-v$(LATEST)'
	git push
	gh pr create --title 'Actions Runner v$(LATEST)'
endif
.PHONY: continuous-upgrade

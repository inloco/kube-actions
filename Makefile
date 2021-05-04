SHELL = /bin/bash
.SHELLFLAGS = -o pipefail -ec

dind/% operator/% runner/%:
	$(MAKE) -C $(@D) $(@F)

.ONESHELL:
continuous-upgrade:
	@
	LATEST="$$(curl -Lf https://api.github.com/repos/actions/runner/releases/latest | jq -r '.tag_name | ltrimstr("v")')"
	if grep -qE "^RUNNER_VERSION \?= $${LATEST}$$" ./runner/Makefile
	then
		echo 'Everything up-to-date'
	elif git ls-remote --exit-code "$$(git remote show)" "feat/actions-runner-v$${LATEST}"
	then
		echo 'Pending merge'
	else
		sed -Ei "s/^AGENT_VERSION \?= .*/AGENT_VERSION ?= $${LATEST}/" ./operator/Makefile
		sed -Ei "s/^RUNNER_VERSION \?= .*/RUNNER_VERSION ?= $${LATEST}/" ./runner/Makefile
		git add -A
		git commit -m "feat: actions/runner#v$${LATEST}"
		git checkout -b "feat/actions-runner-v$${LATEST}"
		git push -u "$$(git remote show)" "$$(git branch --show-current)"
		gh pr create -t "Actions Runner v$${LATEST}" -b "https://github.com/actions/runner/releases/tag/v$${LATEST}"
	fi
.PHONY: continuous-upgrade

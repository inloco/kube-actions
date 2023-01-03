SHELL = /bin/bash
.SHELLFLAGS = -o pipefail -ec

.ONESHELL:
continuous-upgrade:
	@
	echo 'Checking for updates'
	LATEST="$$(curl -Lsf https://api.github.com/repos/actions/runner/releases/latest | jq -r '.tag_name | ltrimstr("v")')"
	echo 'Latest version is' $$LATEST
	if grep -qE "^RUNNER_VERSION \?= $${LATEST}$$" ./runner/Makefile
	then
		echo 'Everything up-to-date'
	elif git ls-remote --exit-code origin "feat/actions-runner-v$${LATEST}"
	then
		echo 'Pending merge'
	elif [ "$$(git ls-remote origin master | cut -f 1)" != "$$(git rev-parse master)" ]
	then
		echo 'Master already changed'
	else
		echo 'Creating update PR'
		sed -Ei "s/^AGENT_VERSION \?= .*/AGENT_VERSION ?= $${LATEST}/" ./operator/Makefile
		sed -Ei "s/^RUNNER_VERSION \?= .*/RUNNER_VERSION ?= $${LATEST}/" ./runner/Makefile
		git add -A
		git commit -m "feat: actions/runner#v$${LATEST}"
		git checkout -b "feat/actions-runner-v$${LATEST}"
		git push -u "$$(git remote show)" "$$(git branch --show-current)"
		gh pr create -t "Actions Runner v$${LATEST}" -b "https://github.com/actions/runner/releases/tag/v$${LATEST}"
	fi
.PHONY: continuous-upgrade

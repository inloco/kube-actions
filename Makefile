SHELL = /bin/bash
.SHELLFLAGS = -o pipefail -ec

.ONESHELL:
continuous-upgrade:
	@
	echo 'Checking for updates'
	LATEST="$$(curl -sSLf --retry 5 https://api.github.com/repos/actions/runner/releases/latest | jq -r '.tag_name | ltrimstr("v")')"
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

		git push -fu "$$(git remote show)" "$$(git branch --show-current):feat/actions-runner-v$${LATEST}"

		cat <<-EOF | gh api graphql -F query=@-
		mutation {
			createCommitOnBranch(
				input: {
					branch: {
					  repositoryNameWithOwner: "inloco/kube-actions",
					  branchName: "feat/actions-runner-v$${LATEST}",
					},
					fileChanges: {
					  additions: [$(
						for F in $(git diff --cached --name-only)
						do
						  printf '{path:"%s",contents:"%s"},' "${F}" "$(base64 ${F} | sed -e ':a' -e '$!N' -e '$!ba' -e 's/\n//g')"
						done
					  )],
					},
					message: {
					  headline: "feat: actions/runner#v$${LATEST}",
					},
					expectedHeadOid: "$(git rev-parse HEAD)",
				},
			) {
				commit {
					oid
				}
			}
		}
		EOF

		gh pr create -t "Actions Runner v$${LATEST}" -b "https://github.com/actions/runner/releases/tag/v$${LATEST}" -H "feat/actions-runner-v$${LATEST}"
	fi
.PHONY: continuous-upgrade

# clover2-cli

Clover2 platform manager: release pipeline (project versions, artifacts) and robot fleet management. One static Go binary.

## Install

Download a binary from [releases](https://github.com/klever-coex/clover2-cli/releases)
(`clover2-cli-linux-amd64`, `clover2-cli-linux-arm64`), put it
on PATH and make it executable:

```bash
curl -fsSL https://github.com/klever-coex/clover2-cli/releases/latest/download/install.sh | sh
```

Or install the deb package - it also installs shell completions and the
`clover2-agent@.service` template (the service is never started automatically):

```bash
sudo dpkg -i clover2-cli_0.1.0_linux_arm64.deb
sudo systemctl enable --now clover2-agent@pi   # run the agent as user pi
```

## Usage

```bash
clover2 version status
clover2 version bump rc
clover2 artifact push build/app.tar.gz
```

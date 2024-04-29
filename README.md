# doltlab-issues

DoltLab is currently closed source.
This repo is used for tracking issues, publishing release notes, and tracking some auxillary DoltLab tools and scripts.

For more general DoltLab information, check out [DoltLab's documentation site](https://docs.doltlab.com) or DoltHub's [blog](https://www.dolthub.com/blog).

# Installer scripts

We've written some scripts to make installing DoltLab's dependencies easier.

- [Install dependencies on Ubuntu](./scripts/ubuntu_install.sh)
- [Install dependencies on CentOS](./scripts/centos_install.sh)

# Tools

Included in DoltLab's releases are some helpful tools written in `go`. You can find the source for these tools in the `go/cmd` package.

- [smtp_connection_helper](./go/cmd/smtp_connection_helper/main.go).

#!/bin/bash

set -e
set -o pipefail

version=""

if [ "$#" -eq 1 ]; then
    echo ""
    echo "Preparing to install DoltLab $1"
    echo ""
    version="$1"
else
    echo ""
    echo "Preparing to install the latest DoltLab"
    echo ""
    version="latest"
fi

with_sudo="sudo"
user="$(whoami)"

export USER=${USER-$user}
export PATH=$PATH:/usr/local/bin

export DEBIAN_FRONTEND=noninteractive

eval sudo -V > /dev/null 2>&1 || with_sudo="" && true

# download doltlab
curl -LO https://doltlab-releases.s3.amazonaws.com/linux/amd64/doltlab-"$version".zip

# create docker group if it doesnt exist
group=docker
eval "$with_sudo getent group $group" || eval "$with_sudo groupadd $group"

# do this here to avoid 'newgrp' command
# which doesnt work well in scripts
if [ $(id -gn) != $group ]; then
  eval exec "$with_sudo" sg $group "$0 $*"
fi

echo "Preparing to download DoltLab $version"

# install tools make, unzip, git
eval "$with_sudo yum makecache -y"
eval "$with_sudo yum install unzip make git -y"

# install docker and docker-compose
eval "$with_sudo yum install -y yum-utils"
eval "$with_sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo"
eval "$with_sudo yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin"
eval "$with_sudo systemctl start docker"

# sanity check
docker --version

# sanity check
compose_command="docker compose"
eval "$compose_command" version > /dev/null 2>&1 || compose_command="docker-compose" && true
eval "$compose_command version"

eval "$with_sudo usermod -aG docker $USER"

# sanity check
docker ps

# install creds-helper and create config
git clone https://github.com/awslabs/amazon-ecr-credential-helper.git
cd amazon-ecr-credential-helper && make docker
eval "$with_sudo mv ./bin/local/docker-credential-ecr-login /usr/local/bin/"

docker-credential-ecr-login -v
cd .. && mkdir -p ~/.docker
echo '{"credHelpers":{"public.ecr.aws":"ecr-login"}}' > ~/.docker/config.json

# unzip DoltLab
unzip doltlab-"$version".zip -d doltlab

echo ""
echo ""
echo "All dependencies installed successfully"
echo ""
echo "DoltLab $version has been download and unzipped to: ./doltlab"
echo ""
echo "Please run 'sudo newgrp docker' to use docker without 'sudo'"
echo ""

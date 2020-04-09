#!/bin/bash
# for ssh access:
# $ adduser github
# $ visudo
# github ALL = NOPASSWD: /path/to/deploy.sh

cd $(dirname "$0")

git reset --hard
git fetch origin
git checkout master
git merge $1

# deploy
docker stack deploy -c <(docker-compose config) notifi-backend
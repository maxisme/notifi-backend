#!/bin/bash
# for ssh access:
# $ adduser github
# $ visudo
# github ALL = NOPASSWD: /path/to/deploy.sh

docker stack deploy -c <(docker-compose config) notifi-backend
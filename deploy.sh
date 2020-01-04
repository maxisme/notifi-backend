#!/bin/bash
# for jenkins ssh:
# $ visudo
# jenk ALL = NOPASSWD: /bin/bash /root/notifi-backend/deploy.sh

cd $(dirname "$0")

git fetch &> /dev/null
diffs=$(git diff master origin/master)

if [ ! -z "$diffs" ]
then
    echo "Pulling code from GitHub..."
    git checkout master
    git pull origin master

    # update app
    docker-compose build app
    docker-compose up --no-deps -d app

    # kill all unused dockers
    docker system prune -f
else
    echo "Already up to date"
fi
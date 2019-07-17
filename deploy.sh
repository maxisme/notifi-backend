#!/bin/bash

cd /root/notifi-backend/

git fetch &> /dev/null
diffs=$(git diff master origin/master)

if [ ! -z "$diffs" ]
then

    echo "Pulling code from GitHub..."
    git checkout master
    git pull origin master

    # update server
    docker-compose up --build -d

    # kill all unused dockers
    docker system prune -f
else
    echo "Already up to date"
fi
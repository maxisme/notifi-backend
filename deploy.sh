#!/bin/bash
# for ssh access:
# $ adduser github
# $ visudo
# github ALL = NOPASSWD: /path/to/deploy.sh

if [[ -z "$1" ]]
then
    echo "No commit sha $@"
    exit 1
fi

# insure only deploying one at a time
DEPLOY_FILE="/tmp/deploying.txt"
while [ -f $DEPLOY_FILE ]
do
    echo "Waiting for another deploy to finish..."
    sleep 1
done

touch $DEPLOY_FILE
trap $(rm -f $DEPLOY_FILE)

cd $(dirname "$0")

git fetch &> /dev/null
diffs=$(git diff master origin/master)

if [[ ! -z "$diffs" ]]
then
    echo "Pulling code from GitHub..."
    git fetch origin
    git checkout master
    git merge $1

    # update schema (-database arg came from docker-compose)
    if ! migrate -source=file://sql/ -database mysql://notifi:notifi@/notifi up
    then
        echo "Failed to run sql migration"
        exit 1
    fi

    # update app
    if ! docker-compose build app
    then
        echo "Failed to build app!"
        exit 1
    fi
    docker-compose up --no-deps -d app

    # kill all unused dockers
    docker system prune -f
else
    echo "Already up to date"
fi

rm -f $DEPLOY_FILE
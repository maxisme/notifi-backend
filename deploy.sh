#!/bin/bash
# for jenkins ssh:
# $ visudo
# jenk ALL = NOPASSWD: /bin/bash /root/notifi-backend/deploy.sh

if [[ -z "$1" ]]
then
    echo "Please add commit sha."
fi

# insure only deploying one at a time
DEPLOY_FILE="/tmp/deploying.txt"
while [ ! -f $DEPLOY_FILE ]
do
    echo "Waiting for another deploy to finish..."
    sleep 1
done
touch $DEPLOY_FILE
trap $(rm -f $DEPLOY_FILE)

cd $(dirname "$0")

git fetch &> /dev/null
diffs=$(git diff master origin/master)

if [ ! -z "$diffs" ]
then
    echo "Pulling code from GitHub..."
    git fetch origin
    git checkout master
    git merge $1

    # update app
    docker-compose build app
    docker-compose up --no-deps -d app

    # kill all unused dockers
    docker system prune -f
else
    echo "Already up to date"
fi

rm -f $DEPLOY_FILE
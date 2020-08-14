#!/bin/bash
git pull

# master
#export ENV_FILE=".env"
#export $(grep -v '^#' $ENV_FILE | xargs)
#docker stack deploy -c stack.yml transfermeit-backend

# dev
export ENV_FILE=".dev.env"
export $(grep -v '^#' $ENV_FILE | xargs)
docker stack deploy -c stack.yml dev-transfermeit-backend
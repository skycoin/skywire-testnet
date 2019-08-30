#!/usr/bin/env bash

body='{
"request": {
    "branch":"'$SWC_BRANCH_NAME'",
    "message":"'$TRIGGER_MESSAGE'", 
    "config": {  
      "merge_mode": "merge",
      "install": "DOCKER_COMPOSE_VERSION=1.24.1 ci_scripts/install-docker-compose.sh;",
      "script": "make vendor-integration-check",      
      "deploy":{
        "provider": "script",
        "script": "echo disabled"
      }
    }
}}'

curl -s -X POST \
   -H "Content-Type: application/json" \
   -H "Accept: application/json" \
   -H "Travis-API-Version: 3" \
   -H "Authorization: token $TRAVIS_API_TOKEN" \
   -d "$body" \
   https://api.travis-ci.com/repo/watercompany%2Fskywire-services/requests
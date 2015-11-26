#!/bin/bash
scriptpath=$(realpath "$0")
stagingpath=$(dirname "$scriptpath")
buildpath=$(dirname "$stagingpath")
mainpath=$(dirname "$buildpath")
zip -j -r "$mainpath/posty-staging.zip" "$stagingpath/Dockerfile" "$stagingpath/Dockerrun.aws.json" "$mainpath/posty" 
cd "$mainpath"
zip -r "./posty-staging.zip" "./frontend/dist" 

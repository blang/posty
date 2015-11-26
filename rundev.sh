#!/bin/bash
exec docker run --rm --env-file="./environment_dev" -p 8080:8080 blang/posty-ebs

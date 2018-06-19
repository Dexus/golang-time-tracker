#!/bin/bash

set -x

for i in $(seq 3); do
  curl -X POST http://localhost:10101/tick --data-binary '{ "labels": ["a"] }'
  sleep 1
done

curl http://localhost:10101/intervals

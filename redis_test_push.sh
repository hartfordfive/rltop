#!/bin/bash

while true; do

  RND1=$(( ( RANDOM % 10 )  + 1 ))
  RND2=$(( ( RANDOM % 10 )  + 1 ))
  START=1

  for (( i=$START; i<=$RND1; i++ ))
  do
    echo "Pushing item $i of $RND1"
    redis-cli rpush another-test-list "{\"beat\":{\"hostname\":\"default-ubuntu-1404\"},\"cis_nginx_ingester\":\"dev-mt-ngx-ingester-09ed511444afd9e81-us-east-1c\",\"@timestamp\":\"2016-10-12T15:48:05.006Z\",\"team\":\"devtools\",\"type\":\"undefined\",\"@filebeat_event_timestamp\":\"2016-10-12T15:49:21Z\",\"message\":\"this is a test 2\",\"environment\":\"production\", \"tags\": [\"test\",\"redis\"]}"
  done 
  
  for (( i=$START; i<=$RND2; i++ ))
  do
    echo "Pushing item $i of $RND2"
    redis-cli rpush test-list "{\"beat\":{\"hostname\":\"default-ubuntu-1404\"},\"cis_nginx_ingester\":\"dev-mt-ngx-ingester-09ed511444afd9e81-us-east-1c\",\"@timestamp\":\"2016-10-12T15:48:05.006Z\",\"team\":\"devtools\",\"type\":\"undefined\",\"@filebeat_event_timestamp\":\"2016-10-12T15:49:21Z\",\"message\":\"this is a test 2\",\"environment\":\"production\", \"tags\": [\"test2\",\"redis2\"]}"
  done 

  sleep $(( ( RANDOM % 3 )  + 1 ))

done

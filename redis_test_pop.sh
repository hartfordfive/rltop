#!/bin/bash

while true; do

  RND1=$(( ( RANDOM % 10 )  + 1 ))
  RND2=$(( ( RANDOM % 10 )  + 1 ))
  START=1

  #for i in {$START..$RND1}
  for (( i=$START; i<=$RND1; i++ ))
  do
    echo "Removing item $i of $RND1"
    redis-cli rpop another-test-list 
  done 
  
  #for i in {$START..$RND2}
  for (( i=$START; i<=$RND2; i++ ))
  do
    echo "Removing item $i of $RND2"
    redis-cli rpop test-list 
  done 

  echo "Sleeping"
  sleep $(( ( RANDOM % 2 )  + 1 ))

done
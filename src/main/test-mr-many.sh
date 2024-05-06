#!/usr/bin/env bash

if [ $# -ne 1 ]; then
    echo "Usage: $0 numTrials"
    exit 1
fi

trap 'kill -INT -$pid; exit 1' INT

# Note: because the socketID is based on the current userID,
# ./test-mr.sh cannot be run in parallel
runs=$1
chmod +x test-mr.sh

success=0
for i in $(seq 1 $runs); do
    timeout -k 2s 900s ./test-mr.sh &
    pid=$!
    if wait $pid; then
        success=$((success+1))
    else
        echo '***' FAILED TESTS IN TRIAL $i
        exit 1
    fi
    echo "成功次数: $success / 总次数: $i"
done
echo '***' PASSED ALL $i TESTING TRIALS
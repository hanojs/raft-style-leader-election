#!/bin/bash
# publish.sh
trap "exit" INT TERM ERR
trap "kill 0" EXIT
go build
for i in "$@"
do
case $i in
    -c|--crash)
    CRASH=true
    ;;
    -n=*|--numNodes=*)
    NUMNODES="${i#*=}"
    ;;
    -p=*|--startingPort=*)
    STARTINGPORT="${i#*=}"
    ;;
    -s|--stdio)
    STDIO=true
    ;;
esac
done

numServers="-numServers=${NUMNODES}"
portBlockStart="-portBlockStart=${STARTINGPORT}"

if [ "$STDIO" = true ]; then

    if [ "$CRASH" = true ]; then
        for i in `seq 1 $NUMNODES`; do
            let portNum=$STARTINGPORT+$i-1
            port="-port=${portNum}"
            sleep 0.5
            ./MP2.exe $numServers $portBlockStart $port "-killLeader" > file${i}.txt 2>&1 &
        done
    else
        for i in `seq 1 $NUMNODES`; do
            let portNum=$STARTINGPORT+$i-1
            port="-port=${portNum}"
            sleep 0.5
            ./MP2.exe $numServers $portBlockStart $port > file${i}.txt 2>&1 &
        done
    fi
else
    if [ "$CRASH" = true ]; then
        for i in `seq 1 $NUMNODES`; do
            let portNum=$STARTINGPORT+$i-1
            port="-port=${portNum}"
            sleep 0.5
            ./MP2.exe $numServers $portBlockStart $port "-killLeader" &
        done
    else
        for i in `seq 1 $NUMNODES`; do
            let portNum=$STARTINGPORT+$i-1
            port="-port=${portNum}"
            sleep 0.5
            ./MP2.exe $numServers $portBlockStart $port &
        done
    fi
fi
wait
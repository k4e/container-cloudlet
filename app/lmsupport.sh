#!/bin/sh
# Line endings must be LF; CRLF cause some error.
main_pid=$1
echo "Main pid: ${main_pid}"
echo -n ${main_pid} > /MAIN_PID
while true; do sleep 60; done

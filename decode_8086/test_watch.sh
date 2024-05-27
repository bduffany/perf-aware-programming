#!/usr/bin/env bash
exec godemon --clear bash -c '
  sleep 0.05
  ./decode_8086/test.sh
'

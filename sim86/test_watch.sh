#!/usr/bin/env bash
exec godemon --clear bash -c '
  sleep 0.05
  ./sim86/test.sh
'

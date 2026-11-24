#!/bin/sh
set -eu

for selected_path in "$@"; do
  if [ -d "$selected_path" ]; then
    clip --smart --recursive "$selected_path"
  else
    clip --smart "$selected_path"
  fi
done

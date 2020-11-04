#!/bin/bash

bazel build //chain/block/genesis/devnet/...

genesis_list=()
while IFS= read -d $'\0' -r file; do
    genesis_list=("${genesis_list[@]}" "$file")
done < <(find -L $(bazel info bazel-bin)/chain/block/genesis/devnet -type f -regextype sed -regex ".*go$" -print0)

arraylength=${#genesis_list[@]}
searchstring="bin/"

for ((i = 0; i < ${arraylength}; i++)); do
    destination=${genesis_list[i]#*$searchstring}
    chmod 755 "$destination"
    cp -R -L "${genesis_list[i]}" "$destination"
done

#!/bin/bash

bazel build //protos/...

proto_list=()
while IFS= read -d $'\0' -r file; do
    proto_list=("${proto_list[@]}" "$file")
done < <(find -L $(bazel info bazel-bin)/protos -type f -regextype sed -regex ".*pb\.\(gw\.\)\?go$" -print0)

arraylength=${#proto_list[@]}
searchstring="theQRL/zond/"

for ((i = 0; i < ${arraylength}; i++)); do
    destination=${proto_list[i]#*$searchstring}
    chmod 755 "$destination"
    cp -R -L "${proto_list[i]}" "$destination"
done

goimports -w protos/**/*.pb.go
gofmt -s -w protos/

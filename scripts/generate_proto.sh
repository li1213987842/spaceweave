#!/usr/bin/env bash

set -e

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  bin="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$bin/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
bin="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
cd $bin


function clean() {
  find .. -name *.pb.go | xargs rm
}

function build() {
  cd ..

  SPACEWEAVE_PROTOCOL_DIR=$(go list -f '{{ .Dir }}' -m github.com/li1213987842/spaceweave)

  for x in $(find . -name "*.proto"); do \
    echo "${x}: Generating Protobuf..."
    protoc \
      -I. \
      -I=${SPACEWEAVE_PROTOCOL_DIR} \
      --go_out=paths=source_relative:. \
      --go-grpc_out=require_unimplemented_servers=false,paths=source_relative:. \
      --go-json_out=emit_defaults=true,orig_name=true,allow_unknown=true,paths=source_relative:. \
      ${x}
    if [ $? -ne 0 ]; then
      echo "Failed, abort"
      exit 1
    fi
  done
}


ACTION=$1

if [ -z $ACTION ]; then
  ACTION='BUILD'
fi

#. config.sh

if [ $ACTION = "CLEAN" ]; then
  clean
else
  build
fi

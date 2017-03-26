#!/bin/sh
# This file is part of the Diamond Construct.
# Copyright (c) 2017, aerth <aerth@riseup.net>
# MIT License

# This script compiles the diamond-admin program and a basic diamond server.
# It can be replaced or adapted to suit your needs.
# Add your target function below

echo '
       *
      ***
     *****
    *******
   *********
  ***********
 *************
***************
 *************
  ***********
   *********
    *******
     *****       Welcome to the Diamond construct.
      ***
       *
'
# ensure we are being ran as 'bin/build.sh'
if [ "$0" != "build/make.sh" ]; then
echo "Environment Error"
echo "Please change directory to: \"\$GOPATH/src/github.com/aerth/diamond\""
echo "Build diamond tools using the command: \"build/make.sh\""
exit 1
fi

# ensure gopath is set
if [ -z "$GOPATH" ]; then
echo "No GOPATH set. Try: GOPATH=/tmp/gopath ./build.sh"
exit 2
fi

# download dependencies
echo Downloading dependencies to $GOPATH/src/github.com/aerth/diamond
go get -v -d github.com/aerth/diamond/...

# exit on error
set -e

# preserve working dir
owd=$PWD

# DIAMOND path to library
if [ -z $DIAMOND  ]; then
DIAMOND=$GOPATH/src/github.com/aerth/diamond/
fi
cd $DIAMOND
echo Building in $DIAMOND

#####
# Define target functions here
#####

# 'bin/build.sh test'
test() {
	cd $DIAMOND/lib && go test -v
}
# 'bin/build.sh admin'
build_admin() {
        cd $DIAMOND/cmd/diamond-admin && make && mkdir -p $DIAMOND/bin && mv diamond-admin $DIAMOND/bin/diamond-admin
}
# 'bin/build.sh server'
build_server() {
       cd $DIAMOND/cmd/diamondd/ && make && mkdir -p $DIAMOND/bin && mv diamondd $DIAMOND/bin/diamondd
}
# 'bin/build.sh custom'
build_custom() {
       cd $DIAMOND/cmd/diamondd/ && make && mkdir -p $DIAMOND/bin && mv diamondd $DIAMOND/bin/diamondd
}


# Display MIT license
cat LICENSE.md
sleep 2

# Switch on user target
CMD=$1
if [ -z "$1" ]; then
CMD="all" # default target: all
fi
echo Building target: $CMD
case $CMD in
'test')
	test
;;
'all')
        echo "⋄ Building ./diamondd and ./diamond-admin"
        echo "⋄ Building diamond-admin"
        build_admin
        echo "⋄ Building diamondd server"
        build_server
 ;;
'admin')
        echo "⋄ Building diamond-admin"
        sleep 1

        build_admin
 ;;
'custom') # rename this target
        echo "⋄ Building diamond server"
        sleep 1
        build_custom
 ;;
'server')
        echo "⋄ Building diamondd server"
        build_server
 ;;
*)
# unknown target
echo "Available targets: $0 all" or "$0 server" or "$0 admin"
;;
esac

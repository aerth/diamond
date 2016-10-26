#!/bin/sh
go get -v -d github.com/aerth/diamond/...
set -e

if [ -z "$GOPATH" ]; then
echo "No GOPATH set. Try: GOPATH=/tmp/gopath ./build.sh"
exit 2
fi


# preserve working dir
wd=$PWD

test() {
	cd $wd/../lib && go test -v
}

build_admin() {
        cd $wd/../cmd/diamond-admin && make && mv diamond-admin $wd/diamond-admin

}

build_server() {
       cd $wd/../cmd/diamondd/ && make && mv diamondd $wd/diamondd 
}



if [ -z $DIAMOND  ]; then
DIAMOND="../lib"
fi

cd $DIAMOND
echo $PWD
lib=$(ls | grep -v lib)
cat LICENSE.md
sleep 1
CMD=$1

if [ -z "$1" ]; then
CMD="all"
fi
echo $CMD

case $CMD in
'test')
	test
;;
'all')
        echo "Building ./diamondd and ./diamond-admin"
        echo "Building diamond-admin"
        build_admin
        echo "Building diamondd server"
        build_server
 ;;
'admin')
        echo "Building diamond-admin"
        sleep 1

        build_admin
 ;;
'server')
        echo "Building diamondd server"
        build_server
 ;;
*)
echo "$0 all" or "$0 server" or "$0 admin"
;;
esac

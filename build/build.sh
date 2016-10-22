#!/bin/sh
set -e

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
ls
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
        echo "COMBO!"
        echo "Building diamond-admin"
        sleep 1
        build_admin
        sleep 1
        echo "Building diamondd server"
        sleep 1
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
esac

#!/bin/bash

do_test_all(){
	for ((i=0;i<5;i++))
	do 
		go test -run 'TestSnapshotUnreliableRecoverConcurrentPartition3B' > bbb$1_$i.log &
	done
}
do_test_all 0 &
do_test_all 1 &
do_test_all 2 &
do_test_all 3 &
do_test_all 4 &
do_test_all 5 &
do_test_all 6 &
do_test_all 7 &
do_test_all 8 &
do_test_all 9 &




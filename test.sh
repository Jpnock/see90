#!/bin/bash

set -euo pipefail

make
./bin/c_compiler -S "./test/all/main.c" -o "./test/all/main.s"

mips-linux-gnu-gcc -mfp32 -o ./test/all/main.o -c ./test/all/main.s
mips-linux-gnu-gcc -mfp32 -static -o ./test/all/main ./test/all/main.o ./test/all/root.c
docker build -t see90 .
docker run -v "$(pwd)/test/all":"/mnt/test" see90 /mnt/test/test.sh
rm ./test/all/main
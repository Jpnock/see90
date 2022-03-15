#!/bin/bash

set -uo pipefail

shopt -s globstar

make
docker build -t see90 .

for f in test/compiler_tests/**/*_driver.c; do
    assemble="${f%_driver.c}.c"

    echo "Running test: ${f}"

    ./bin/see90 < "$assemble" > ./test/compiler_tests/main.s
    mips-linux-gnu-gcc -mfp32 -o ./test/compiler_tests/main.o -c ./test/compiler_tests/main.s
    mips-linux-gnu-gcc -mfp32 -static -o ./test/compiler_tests/main ./test/compiler_tests/main.o "$f"
    docker run -v "$(pwd)/test/compiler_tests":"/mnt/test" -p 54321:54321 see90 /mnt/test/test.sh
    rm ./test/compiler_tests/main
    rm ./test/compiler_tests/main.o
    # rm ./test/compiler_tests/main.s
done

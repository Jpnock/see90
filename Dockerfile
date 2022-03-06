FROM ubuntu:focal

RUN apt-get update
RUN apt-get -y install g++-mips-linux-gnu gdb-multiarch qemu qemu-user

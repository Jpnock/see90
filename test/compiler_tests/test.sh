#!/bin/bash

set -uo pipefail

cd /mnt/test || exit 123
qemu-mips main
echo "Returned: $?"

#!/bin/bash

set -uo pipefail

cd /mnt/test || exit 255
qemu-mips main
echo "Returned: $?"

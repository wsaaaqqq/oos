#!/bin/bash
set -e

BIN_DIR="${HOME}/.local/bin"

if [ -f "${BIN_DIR}/oos" ]; then
  rm "${BIN_DIR}/oos"
  echo "oos removed from ${BIN_DIR}"
else
  echo "oos not found in ${BIN_DIR}"
fi

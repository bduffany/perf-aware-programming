#!/usr/bin/env bash
set -e

# Test suite - run it from the root directory like
# $ ./decode_8086/test.sh
#
# Or run it interactively:
# $ ./decode_8086/test_watch.sh

cd "$(dirname "$0")"

rm -rf out && mkdir -p out

go build -o ./out/ decode.go

bindump() {
  ../bindump
}

FILE_WIDTH=56

print_result() {
  local file="$1"
  local style="$2"
  local result="$3"
  printf "%-${FILE_WIDTH}s\t\x1b[${style}m%s\x1b[m\n" "$file" "$result" >&2
}

for FILE in *.asm; do
  # Assemble with nasm
  BIN="./out/${FILE/.asm/}"
  nasm "$FILE" -o "$BIN"
  bindump <"$BIN" >"$BIN.bindump"
  # Disassemble with our program
  OK=1
  if ! ./out/decode <"$BIN" >"$BIN.asm" 2>"$BIN.log"; then
    OK=0
  fi
  # Re-assemble our disassembled version then diff against nasm output
  DIFF_CONTENT=""
  if ((OK)); then
    nasm "$BIN.asm" -o "${BIN}_reassembled"
    bindump <"${BIN}_reassembled" >"${BIN}_reassembled.bindump"
    DIFF_CONTENT=$(diff -u "$BIN.bindump" "${BIN}_reassembled.bindump" | colordiff)
  fi
  if ! [[ "$DIFF_CONTENT" ]] && ((OK)); then
    print_result "$FILE" '32;1' 'PASSED'
  else
    print_result "$FILE" '31;1' 'FAILED'
    cat "$BIN.log" >&2
  fi
  printf '%s' "$DIFF_CONTENT"
done

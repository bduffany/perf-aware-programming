#!/usr/bin/env bash
set -e

# Test suite - run it from the root directory like
# $ ./sim86/test.sh
#
# Or run it interactively:
# $ ./sim86/test_watch.sh

cd "$(dirname "$0")"

rm -rf out && mkdir -p out

go build -o ./out/ *.go

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

test_decode() {
  LISTING_NUMBERS=(
    '0037'
    '0038'
    '0039'
    '0041'
  )
  for NUMBER in "${LISTING_NUMBERS[@]}"; do
    FILE=$(echo ../computer_enhance/perfaware/part1/listing_"${NUMBER}"_*.asm)
    BASE=$(basename "$FILE")

    # Assemble with nasm
    BIN="./out/${BASE/.asm/}"
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
      print_result "$BASE" '32;1' 'PASSED'
    else
      print_result "$BASE" '31;1' 'FAILED'
      cat "$BIN.log" >&2
    fi
    printf '%s' "$DIFF_CONTENT"
  done
}

test_exec() {
  LISTING_NUMBERS=(
    '0043'
    '0044'
  )
  for NUMBER in "${LISTING_NUMBERS[@]}"; do
    FILE=$(echo ../computer_enhance/perfaware/part1/listing_"${NUMBER}"_*.asm)
    BASE=$(basename "$FILE")

    # Assemble with nasm
    BIN="./out/${BASE/.asm/}"
    nasm "$FILE" -o "$BIN"
    bindump <"$BIN" >"$BIN.bindump"
    # Execute with our program
    OK=1
    if ! ./out/decode -exec -name="test\\${BASE/.asm/}" <"$BIN" >"$BIN.txt" 2>"$BIN.log"; then
      OK=0
    fi
    # Diff against expected output
    DIFF_CONTENT=""
    if ((OK)); then
      # Use unix line endings
      tr -d '\r' <../computer_enhance/perfaware/part1/listing_"${NUMBER}"_*.txt >./out/"${BASE/.asm/.expected.txt}"
      DIFF_CONTENT=$(diff -u ./out/"${BASE/.asm/.expected.txt}" "$BIN.txt" | colordiff)
    fi
    if ! [[ "$DIFF_CONTENT" ]] && ((OK)); then
      print_result "$BASE" '32;1' 'PASSED'
    else
      print_result "$BASE" '31;1' 'FAILED'
      cat "$BIN.log" >&2
    fi
    printf '%s' "$DIFF_CONTENT"
  done
}

test_decode
test_exec

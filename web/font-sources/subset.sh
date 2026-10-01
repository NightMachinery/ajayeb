#!/bin/sh
# Rebuild the shipped WOFF2 fonts in web/static/fonts/ from the original TTFs
# kept next to this script. Needs pyftsubset (fonttools) with brotli.
#
#   sh web/font-sources/subset.sh
#
# The subset covers Latin-1, the common punctuation block, arrows, the euro
# and trade marks, and the minus sign. It keeps kerning, standard ligatures
# and tabular figures (the UI sets numbers with tabular-nums).
set -eu

here=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
out="$here/../static/fonts"

# Keep in sync with the unicode-range of the @font-face rules in src/app.css.
unicodes='U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,U+02DC,U+2000-206F,U+20AC,U+2122,U+2190-2199,U+2212,U+2215,U+FEFF,U+FFFD'

command mkdir -p "$out"
for name in fraunces-600 inter-400 inter-600 inter-700; do
  command pyftsubset "$here/$name.ttf" \
    --unicodes="$unicodes" \
    --layout-features='kern,liga,tnum' \
    --flavor=woff2 \
    --output-file="$out/$name.woff2"
  printf '%s\n' "wrote static/fonts/$name.woff2"
done

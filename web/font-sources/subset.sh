#!/bin/sh
# Rebuild the shipped WOFF2 fonts in web/src/lib/fonts/ from the original TTFs
# kept next to this script. Needs pyftsubset (fonttools) with brotli. The
# @font-face rules in src/app.css name them, so the build emits them under
# content-hashed URLs in /_app/immutable/.
#
#   sh web/font-sources/subset.sh
#
# Fraunces and Inter: the subset covers Latin-1, the common punctuation
# block, arrows, the euro and trade marks, and the minus sign. It keeps
# kerning, standard ligatures and tabular figures (the UI sets numbers with
# tabular-nums).
#
# Vazirmatn (Persian): the Arabic block (letters, harakat, Persian and
# Arabic-Indic digits, the Arabic comma, semicolon, question mark, percent
# and decimal and thousands separators), ZWNJ and ZWJ, and the presentation
# forms, nothing Latin, so Latin text keeps Inter. Not the direction marks
# (U+200E, U+200F): the English interface wraps strings in them, Inter has
# no glyph for them, and a range that claimed them would make every English
# page fetch Vazirmatn for an invisible character. It
# keeps pyftsubset's default layout features, which include the Arabic
# joining forms (init, medi, fina, isol), rlig, ccmp, calt, mark, mkmk and
# kern, plus tabular figures.
#
# The originals are the upstream release v33.003 of Vazirmatn by Saber
# Rastikerdar, github.com/rastikerdar/vazirmatn/releases/tag/v33.003
# (vazirmatn-v33.003.zip, fonts/ttf/Vazirmatn-{Regular,SemiBold,Bold}.ttf,
# renamed vazirmatn-{400,600,700}.ttf).
set -eu

here=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
out="$here/../src/lib/fonts"

# Keep in sync with the unicode-range of the @font-face rules in src/app.css.
unicodes='U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,U+02DC,U+2000-206F,U+20AC,U+2122,U+2190-2199,U+2212,U+2215,U+FEFF,U+FFFD'

# Keep in sync with the unicode-range of the Vazirmatn @font-face rules.
persian='U+0600-06FF,U+200C-200D,U+FB50-FDFF,U+FE70-FEFC'

command mkdir -p "$out"
for name in fraunces-600 inter-400 inter-600 inter-700; do
  command pyftsubset "$here/$name.ttf" \
    --unicodes="$unicodes" \
    --layout-features='kern,liga,tnum' \
    --flavor=woff2 \
    --output-file="$out/$name.woff2"
  printf '%s\n' "wrote src/lib/fonts/$name.woff2"
done

for name in vazirmatn-400 vazirmatn-600 vazirmatn-700; do
  command pyftsubset "$here/$name.ttf" \
    --unicodes="$persian" \
    --layout-features+=tnum \
    --flavor=woff2 \
    --output-file="$out/$name.woff2"
  printf '%s\n' "wrote src/lib/fonts/$name.woff2"
done

#!/usr/bin/env bash
#
# build-corpus.sh: download and convert the Demo Corpus media into library/.
#
# Run it on an empty library/ to rebuild from scratch, then author the database
# with Shisho itself (see demo/README.md in shishobooks/shisho). Upstream files
# are cached in .downloads/ (gitignored) so a re-run only redoes conversions.
# Shisho reorganizes library/ on the first scan, so the layout written here is
# only the starting point: one directory per Book.
#
# Requirements: bash 4+, curl, ffmpeg with libfdk_aac, ImageMagick 7 (magick),
# zip, and bsdtar built with RAR support (macOS and most Linux distributions).
#
# Every source and its license basis is recorded in CORPUS.md. Keep the two in
# sync when adding or replacing a work.

set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

UA="shisho-demo-corpus (https://github.com/shishobooks/demo-corpus)"
DL=.downloads
LIB=${LIB:-library}

# Comic pages are downscaled to this longest edge and JPEG quality.
PAGE_MAX_PX=1600
PAGE_QUALITY=80
# Old newsprint scans compress well; a lower quality keeps each issue small.
SCAN_QUALITY=70

# Audio re-encode settings for the long audiobook (HE-AAC, mono).
AUDIO_BITRATE=24k

mkdir -p "$DL" "$LIB"

log() { printf '%s\n' "$*" >&2; }

# fetch <url> <cache path>
fetch() {
  local url=$1 dest=$2
  if [ -s "$dest" ]; then
    log "cached   $dest"
    return
  fi
  log "download $url"
  mkdir -p "$(dirname "$dest")"
  curl -fL --retry 3 --retry-delay 5 -A "$UA" -o "$dest.part" "$url"
  mv "$dest.part" "$dest"
}

# place <cache path> <book directory> <file name>: copy an unmodified file in.
place() {
  local src=$1 dir="$LIB/$2" name=$3
  mkdir -p "$dir"
  if [ -e "$dir/$name" ]; then
    log "exists   $dir/$name"
  else
    cp "$src" "$dir/$name"
    log "placed   $dir/$name"
  fi
}

# write_comicinfo <dir> <title> <series> <number> <year> <month> <day> <writer> <publisher> <web> <summary>
write_comicinfo() {
  local dir=$1 title=$2 series=$3 number=$4 year=$5 month=$6 day=$7 writer=$8 publisher=$9 web=${10} summary=${11}
  local count
  count=$(find "$dir" -maxdepth 1 -name '*.jpg' | wc -l | tr -d ' ')
  cat > "$dir/ComicInfo.xml" <<XML
<?xml version="1.0" encoding="utf-8"?>
<ComicInfo xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <Title>$(xml_escape "$title")</Title>
  <Series>$(xml_escape "$series")</Series>
  <Number>$number</Number>
  <Summary>$(xml_escape "$summary")</Summary>
  <Year>$year</Year>
  <Month>$month</Month>
  <Day>$day</Day>
$(creator_tags "$writer")
  <Publisher>$(xml_escape "$publisher")</Publisher>
  <Web>$web</Web>
  <PageCount>$count</PageCount>
  <LanguageISO>en</LanguageISO>
</ComicInfo>
XML
}

# creator_tags <name>: Writer and Penciller elements, omitted when unknown.
creator_tags() {
  if [ -n "$1" ]; then
    printf '  <Writer>%s</Writer>\n  <Penciller>%s</Penciller>\n' "$(xml_escape "$1")" "$(xml_escape "$1")"
  fi
}

xml_escape() {
  local s=$1
  s=${s//&/&amp;}
  s=${s//</&lt;}
  s=${s//>/&gt;}
  s=${s//\"/&quot;}
  printf '%s' "$s"
}

# pack_cbz <staging dir> <book directory> <file name>
pack_cbz() {
  local staging=$1 dir="$LIB/$2" name=$3
  mkdir -p "$dir"
  if [ -e "$dir/$name" ]; then
    log "exists   $dir/$name"
    return
  fi
  (cd "$staging" && rm -f out.cbz && zip -q -X -r out.cbz ComicInfo.xml ./*.jpg)
  mv "$staging/out.cbz" "$dir/$name"
  log "packed   $dir/$name"
}

# downscale <input image> <output jpg> [quality]
downscale() {
  magick "$1" -auto-orient -strip -resize "${PAGE_MAX_PX}x${PAGE_MAX_PX}>" -quality "${3:-$PAGE_QUALITY}" "$2"
}

# ---------------------------------------------------------------------------
# EPUB and PDF, used unmodified
# ---------------------------------------------------------------------------

# Open Advice (CC BY-SA 3.0). The official site is HTTP only.
fetch "http://open-advice.org/Open-Advice.epub" "$DL/open-advice/Open-Advice.epub"
fetch "http://open-advice.org/Open-Advice.pdf" "$DL/open-advice/Open-Advice.pdf"
place "$DL/open-advice/Open-Advice.epub" "Open Advice" "Open Advice.epub"
place "$DL/open-advice/Open-Advice.pdf" "Open Advice" "Open Advice.pdf"

# Software Engineering: Standing on the Shoulders of Giants, release 1.0b16
# (CC BY-SA 4.0 prose, MIT code).
SWE_RELEASE="https://github.com/tghastings/open-swe-book/releases/download/1.0b16"
fetch "$SWE_RELEASE/swebook-generic.epub" "$DL/open-swe-book/swebook-generic.epub"
fetch "$SWE_RELEASE/swebook-generic.pdf" "$DL/open-swe-book/swebook-generic.pdf"
place "$DL/open-swe-book/swebook-generic.epub" "Software Engineering - Standing on the Shoulders of Giants" "Software Engineering - Standing on the Shoulders of Giants.epub"
place "$DL/open-swe-book/swebook-generic.pdf" "Software Engineering - Standing on the Shoulders of Giants" "Software Engineering - Standing on the Shoulders of Giants.pdf"

# Made with Creative Commons (CC BY-SA 4.0).
fetch "https://creativecommons.org/wp-content/uploads/2017/04/made-with-cc.pdf" "$DL/made-with-cc/made-with-cc.pdf"
place "$DL/made-with-cc/made-with-cc.pdf" "Made with Creative Commons" "Made with Creative Commons.pdf"

# Standard Ebooks editions (CC0 for the edition; underlying works public domain).
# Standard Ebooks requires ?source=download on direct downloads.
SE="https://standardebooks.org/ebooks"
fetch "$SE/l-frank-baum/the-wonderful-wizard-of-oz/downloads/l-frank-baum_the-wonderful-wizard-of-oz.epub?source=download" \
  "$DL/standardebooks/l-frank-baum_the-wonderful-wizard-of-oz.epub"
fetch "$SE/l-frank-baum/the-marvelous-land-of-oz/downloads/l-frank-baum_the-marvelous-land-of-oz.epub?source=download" \
  "$DL/standardebooks/l-frank-baum_the-marvelous-land-of-oz.epub"
fetch "$SE/oscar-wilde/the-importance-of-being-earnest/downloads/oscar-wilde_the-importance-of-being-earnest.epub?source=download" \
  "$DL/standardebooks/oscar-wilde_the-importance-of-being-earnest.epub"
fetch "$SE/charles-dickens/a-christmas-carol/downloads/charles-dickens_a-christmas-carol.epub?source=download" \
  "$DL/standardebooks/charles-dickens_a-christmas-carol.epub"
place "$DL/standardebooks/l-frank-baum_the-wonderful-wizard-of-oz.epub" "The Wonderful Wizard of Oz" "The Wonderful Wizard of Oz.epub"
place "$DL/standardebooks/l-frank-baum_the-marvelous-land-of-oz.epub" "The Marvelous Land of Oz" "The Marvelous Land of Oz.epub"
place "$DL/standardebooks/oscar-wilde_the-importance-of-being-earnest.epub" "The Importance of Being Earnest" "The Importance of Being Earnest.epub"
place "$DL/standardebooks/charles-dickens_a-christmas-carol.epub" "A Christmas Carol" "A Christmas Carol.epub"

# ---------------------------------------------------------------------------
# M4B audiobooks from LibriVox
# ---------------------------------------------------------------------------

# The Velveteen Rabbit, read by Marlo Dianne. Small enough to keep unmodified.
fetch "https://archive.org/download/velveteen_rabbit_librivox/velveteen_rabbit_librivox.m4b" \
  "$DL/librivox/velveteen_rabbit_librivox.m4b"
place "$DL/librivox/velveteen_rabbit_librivox.m4b" "The Velveteen Rabbit" "The Velveteen Rabbit.m4b"

# The Wonderful Wizard of Oz, 2007 solo reading by J. Hall (3 h 45 min).
# Re-encoded from the 48 kbps stereo original to 24 kbps HE-AAC mono to
# fit the corpus size target. Chapters, tags, and the cover are carried over.
fetch "https://archive.org/download/wizard_of_oz/WonderfulWizardOfOz-48kb_librivox.m4b" \
  "$DL/librivox/WonderfulWizardOfOz-48kb_librivox.m4b"
OZ_OUT="$LIB/The Wonderful Wizard of Oz/The Wonderful Wizard of Oz.m4b"
if [ -e "$OZ_OUT" ]; then
  log "exists   $OZ_OUT"
else
  mkdir -p "$(dirname "$OZ_OUT")"
  ffmpeg -hide_banner -loglevel error -y \
    -i "$DL/librivox/WonderfulWizardOfOz-48kb_librivox.m4b" \
    -map 0:a:0 -map 0:v? -map_metadata 0 -map_chapters 0 \
    -c:a libfdk_aac -profile:a aac_he -ac 1 -b:a "$AUDIO_BITRATE" \
    -c:v copy -disposition:v attached_pic \
    -movflags +faststart -f ipod "$OZ_OUT.part.m4b"
  mv "$OZ_OUT.part.m4b" "$OZ_OUT"
  log "encoded  $OZ_OUT"
fi

# ---------------------------------------------------------------------------
# Pepper&Carrot by David Revoy (CC BY 4.0): English page images packed into CBZ
# ---------------------------------------------------------------------------

PC_CREDIT="Pepper&Carrot by David Revoy, https://www.peppercarrot.com, CC BY 4.0. Pages downscaled and packed into CBZ for the Shisho demo."

# The page-zero header of each episode is a wide title banner, which makes a
# poor portrait cover. build_cover composes one: the episode's own lettering
# on a white band above a 4:5 crop of a text-free panel, at 2:3 overall. It
# becomes the first page of the CBZ, which is where Shisho reads a comic's
# cover from (CBZ covers cannot be replaced by upload).
#
# build_cover <episode number> <slug> <page> <crop WxH+X+Y> <output jpg>
build_cover() {
  local ep=$1 slug=$2 page=$3 crop=$4 out=$5
  local base="https://www.peppercarrot.com/0_sources/$slug/hi-res"
  local gfx="$DL/peppercarrot/ep$ep/gfx-E${ep}P${page}.jpg"
  local staging="$DL/peppercarrot/ep$ep-cover"
  fetch "$base/gfx-only/gfx_Pepper-and-Carrot_by-David-Revoy_E${ep}P${page}.jpg" "$gfx"
  rm -rf "$staging"
  mkdir -p "$staging"
  magick "$DL/peppercarrot/ep$ep/E${ep}P00.jpg" -trim +repage -resize 1040x220 \
    -background white -gravity center -extent 1200x300 "$staging/header.png"
  magick "$gfx" -crop "$crop" +repage -resize 1200x1500^ -gravity center -extent 1200x1500 "$staging/art.jpg"
  magick "$staging/header.png" "$staging/art.jpg" -append -strip -quality 88 "$out"
  rm -rf "$staging"
}

# build_peppercarrot <episode number> <slug> <title> <last page> <year> <month> <day> <cover page> <cover crop>
build_peppercarrot() {
  local ep=$1 slug=$2 title=$3 last=$4 year=$5 month=$6 day=$7 cover_page=$8 cover_crop=$9
  local base="https://www.peppercarrot.com/0_sources/$slug/hi-res"
  local staging="$DL/peppercarrot/ep$ep-staging"
  local page
  local cbz="$LIB/Pepper and Carrot/Episode $ep - $title/Pepper and Carrot - Episode $ep - $title.cbz"
  if [ -e "$cbz" ]; then
    log "exists   Pepper and Carrot episode $ep"
    return
  fi
  rm -rf "$staging"
  mkdir -p "$staging"
  for page in $(seq -f '%02g' 0 "$last"); do
    fetch "$base/en_Pepper-and-Carrot_by-David-Revoy_E${ep}P${page}.jpg" "$DL/peppercarrot/ep$ep/E${ep}P${page}.jpg"
    downscale "$DL/peppercarrot/ep$ep/E${ep}P${page}.jpg" "$staging/P${page}.jpg"
  done
  # "000-cover" sorts before "P00" inside the archive, so it is page one.
  build_cover "$ep" "$slug" "$cover_page" "$cover_crop" "$staging/000-cover.jpg"
  write_comicinfo "$staging" "$title" "Pepper&Carrot" "$ep" "$year" "$month" "$day" "David Revoy" "David Revoy" \
    "https://www.peppercarrot.com/en/webcomic/$slug.html" "$PC_CREDIT"
  pack_cbz "$staging" "Pepper and Carrot/Episode $ep - $title" "Pepper and Carrot - Episode $ep - $title.cbz"
  rm -rf "$staging"
}

# Cover panels: episode 24 uses the library scene at the top of page 5 (the
# site's own thumbnail for the episode); episode 25 uses the big panel on page 7.
build_peppercarrot 24 "ep24_The-Unity-Tree" "The Unity Tree" 8 2017 12 15 05 "1106x1382+900+0"
build_peppercarrot 25 "ep25_There-are-no-Shortcuts" "There are no Shortcuts" 10 2018 5 28 07 "1135x1418+673+1134"

# ---------------------------------------------------------------------------
# Planet Comics (Fiction House, 1940), United States public domain
# ---------------------------------------------------------------------------

PLANET_ITEM="https://archive.org/download/planet-comics-011-gm-removed-cbpop"
PLANET_CREDIT="Planet Comics, Fiction House, 1940. United States public domain (copyright not renewed). Scans from archive.org, downscaled and repacked for the Shisho demo."

# build_planet <issue number> <archive file name> <year> <month>
build_planet() {
  local issue=$1 remote=$2 year=$3 month=$4
  local padded
  padded=$(printf '%02d' "$issue")
  local archive="$DL/planet-comics/$remote"
  local extracted="$DL/planet-comics/issue-$padded-extracted"
  local staging="$DL/planet-comics/issue-$padded-staging"
  if [ -e "$LIB/Planet Comics/Planet Comics $padded/Planet Comics $padded.cbz" ]; then
    log "exists   Planet Comics $padded"
    return
  fi
  fetch "$PLANET_ITEM/$remote" "$archive"
  rm -rf "$extracted" "$staging"
  mkdir -p "$extracted" "$staging"
  bsdtar -xf "$archive" -C "$extracted"
  local n=0 img
  while IFS= read -r img; do
    n=$((n + 1))
    downscale "$img" "$staging/$(printf '%03d' "$n").jpg" "$SCAN_QUALITY"
  done < <(find "$extracted" -type f \( -iname '*.jpg' -o -iname '*.jpeg' -o -iname '*.png' -o -iname '*.webp' \) | sort)
  write_comicinfo "$staging" "Planet Comics #$issue" "Planet Comics" "$issue" "$year" "$month" 1 "" "Fiction House" \
    "https://archive.org/details/planet-comics-011-gm-removed-cbpop" "$PLANET_CREDIT"
  pack_cbz "$staging" "Planet Comics/Planet Comics $padded" "Planet Comics $padded.cbz"
  rm -rf "$extracted" "$staging"
}

build_planet 1 "Planet_Comics_01__c2c.cbr" 1940 1
build_planet 3 "Planet_Comics_003_JVJon.cbr" 1940 3
build_planet 5 "Planet_Comics__05__paper__68pg_c2c.cbr" 1940 5

log ""
log "Corpus size:"
du -sh "$LIB" >&2

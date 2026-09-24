# Shisho Demo Corpus

The media, covers, sidecars, and prepared database served by the Shisho Public Demo at `https://demo.shishobooks.com`.

- `library/`: the Books as Shisho organizes them, plus generated covers and sidecars.
- `config/shisho.db`: the prepared database. `config/cache/` is gitignored.
- `CORPUS.md`: per-work source, license basis, credit, and modifications. Read it before adding anything.
- `build-corpus.sh`: reproduces the downloads and conversions.

A push to `master` triggers a demo deploy in `shishobooks/shisho`. The authoring loop is documented in that repository's `demo/README.md`.

# Demo Corpus credits

This repository is the Demo Corpus for the Shisho Public Demo at `https://demo.shishobooks.com`. This file is the canonical credits page: every work in `library/`, where it came from, why it may be redistributed, the credit its license asks for, and what was changed. `build-corpus.sh` reproduces the downloads and conversions.

## Clearance policy

A work may enter the corpus when either (1) the rights holder published it under an explicit license permitting commercial redistribution and adaptation (CC BY, CC BY-SA, CC0), or (2) the underlying work's author and illustrator died more than 70 years ago and the digital edition's contributors dedicate their own work to the public domain, or the work is a pre-1956 United States publication whose copyright was not renewed. NonCommercial works are excluded. Modified copies (re-encoded audio, downscaled pages) are marked as modified below. This is a pragmatic policy for a free promotional demo, not a worldwide legal guarantee. Public-domain status can differ by country; if you believe a work is included in error, open an issue and it will be removed.

Names and marks of the projects credited below are used only to identify the works. None of them sponsor or endorse Shisho.

## Works

### Open Advice: FOSS: What We Wish We Had Known When We Started

- Files: EPUB, PDF
- Source: <http://open-advice.org/> (files `Open-Advice.epub` and `Open-Advice.pdf`), source repository <https://github.com/Open-Advice/Open-Advice>
- License: CC BY-SA 3.0, <https://creativecommons.org/licenses/by-sa/3.0/>
- Credit: Open Advice, edited by Lydia Pintscher, with the contributors listed in the book. Licensed CC BY-SA 3.0.
- Modifications: none. Files renamed.

### Software Engineering: Standing on the Shoulders of Giants

- Files: EPUB, PDF (generic edition, release 1.0b16)
- Source: <https://github.com/tghastings/open-swe-book/releases/tag/1.0b16>
- License: CC BY-SA 4.0 for prose, figures, and diagrams; MIT for code snippets. <https://github.com/tghastings/open-swe-book/blob/1.0b16/LICENSE>
- Credit: Software Engineering: Standing on the Shoulders of Giants, by the open contributors of the open-swe-book repository, licensed under CC BY-SA 4.0. The project notes that its prose was drafted with AI assistance under author direction and review.
- Modifications: none. Files renamed.

### Made with Creative Commons

- Files: PDF
- Source: <https://creativecommons.org/wp-content/uploads/2017/04/made-with-cc.pdf>
- License: CC BY-SA 4.0, <https://creativecommons.org/licenses/by-sa/4.0/>
- Credit: Made with Creative Commons by Paul Stacey and Sarah Hinchliff Pearson, Creative Commons, 2017. Licensed CC BY-SA 4.0.
- Modifications: none. File renamed.

### The Wonderful Wizard of Oz (Oz, 1)

- Files: EPUB, M4B
- EPUB source: Standard Ebooks, <https://standardebooks.org/ebooks/l-frank-baum/the-wonderful-wizard-of-oz>
- M4B source: LibriVox, 2007 solo recording read by J. Hall, <https://librivox.org/the-wonderful-wizard-of-oz-by-l-frank-baum/> (file `WonderfulWizardOfOz-48kb_librivox.m4b` from <https://archive.org/details/wizard_of_oz>)
- Basis: L. Frank Baum died in 1919 and illustrator W. W. Denslow in 1915, so the 1900 work is in the public domain. Standard Ebooks dedicates its edition to the public domain under CC0. LibriVox dedicates its recordings to the public domain.
- Credit: Text from the Standard Ebooks edition (CC0). Audiobook from LibriVox, read by J. Hall, public domain.
- Modifications: the EPUB is unmodified. The audiobook was re-encoded from 48 kbps stereo AAC to 24 kbps HE-AAC mono with ffmpeg to reduce its size; chapters, tags, and the cover were carried over.

### The Marvelous Land of Oz (Oz, 2)

- Files: EPUB
- Source: Standard Ebooks, <https://standardebooks.org/ebooks/l-frank-baum/the-marvelous-land-of-oz>
- Basis: L. Frank Baum died in 1919 and illustrator John R. Neill in 1943. Standard Ebooks dedicates its edition to the public domain under CC0.
- Credit: Text from the Standard Ebooks edition (CC0).
- Modifications: none. File renamed.

### The Importance of Being Earnest

- Files: EPUB
- Source: Standard Ebooks, <https://standardebooks.org/ebooks/oscar-wilde/the-importance-of-being-earnest>
- Basis: Oscar Wilde died in 1900. Standard Ebooks dedicates its edition to the public domain under CC0.
- Credit: Text from the Standard Ebooks edition (CC0).
- Modifications: none. File renamed.

### A Christmas Carol

- Files: EPUB
- Source: Standard Ebooks, <https://standardebooks.org/ebooks/charles-dickens/a-christmas-carol>
- Basis: Charles Dickens died in 1870. Standard Ebooks dedicates its edition to the public domain under CC0.
- Credit: Text from the Standard Ebooks edition (CC0).
- Modifications: none. File renamed.

### The Velveteen Rabbit

- Files: M4B
- Source: LibriVox, read by Marlo Dianne, <https://librivox.org/the-velveteen-rabbit-by-margery-williams/> (file `velveteen_rabbit_librivox.m4b` from <https://archive.org/details/velveteen_rabbit_librivox>)
- Basis: Margery Williams died in 1944 and illustrator William Nicholson in 1949. LibriVox dedicates its recordings to the public domain.
- Credit: Audiobook from LibriVox, read by Marlo Dianne, public domain.
- Modifications: none. File renamed.

### Pepper&Carrot, episodes 24 and 25

- Files: CBZ (episode 24, The Unity Tree), CBZ (episode 25, There are no Shortcuts). Each episode is its own Book in the Pepper&Carrot Series.
- Source: <https://www.peppercarrot.com/> (English hi-res page images from the episode source folders)
- License: CC BY 4.0, <https://creativecommons.org/licenses/by/4.0/>
- Credit: Pepper&Carrot by David Revoy, <https://www.peppercarrot.com>, licensed CC BY 4.0.
- Modifications: the page images were downscaled to 1600 px on the longest edge, saved as JPEG at quality 80, and packed into CBZ files with a generated `ComicInfo.xml`.

### Planet Comics, issues 1, 3, and 5

- Files: CBZ x3. Each issue is its own Book in the Planet Comics Series.
- Source: scans from <https://archive.org/details/planet-comics-011-gm-removed-cbpop>
- Basis: published by Fiction House in 1940 in the United States; copyright was not renewed, so the issues are in the United States public domain.
- Credit: Planet Comics, Fiction House, 1940. Scans via archive.org.
- Modifications: the scanned pages were downscaled to 1600 px on the longest edge, saved as JPEG at quality 80, and repacked as CBZ files with a generated `ComicInfo.xml`.

## Prepared database

`config/shisho.db` was authored with Shisho itself against `library/` mounted at `/media`. It contains one library at `/media` with 13 Books and 16 Files (the spec's ten works, with comic issues and episodes as separate Books in their Series), a throwaway admin account, and the shared `demo` visitor account. Because this database is public, neither password protects anything:

| User | Role | Password |
|------|------|----------|
| `admin` | Admin | `demo-admin-throwaway` |
| `demo` | Viewer | `shishodemo` |

The Public Demo runs in Demo Mode, where admins have no bypass and every persistent change is rejected. See `demo/README.md` in `shishobooks/shisho` for the authoring loop.

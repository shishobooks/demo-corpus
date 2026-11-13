// Command reading-history writes the demo user's sample reading into the
// prepared database: a year of Read-throughs with Reading Sessions, Ratings,
// and the Reading Positions and position log the web reader would have left,
// with a few Books still underway. It also dates each Book's arrival in the
// library ahead of the reading. plan.go says what was read and added when.
//
// It replaces whatever reading the demo user has, and the same latest day
// writes the same history, so rerun it after the library changes:
//
//	cd reading-history && go run .
//
// Run it with Shisho stopped, on a database already migrated to the release
// the demo runs. In Demo Mode, Shisho moves these dates so the latest
// reading is always yesterday, so the day they end on doesn't matter.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"maps"
	"math"
	"math/rand/v2"
	"path/filepath"
	"slices"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := flag.String("db", "../config/shisho.db", "the prepared database")
	library := flag.String("library", "../library", "the corpus library, which Shisho sees as -media")
	media := flag.String("media", "/media", "where Shisho sees the library")
	username := flag.String("user", "demo", "the user whose reading to write")
	// A fixed day, so rerunning an unchanged plan leaves the database as it
	// was. The weekday patterns line up with this day's weekdays.
	latest := flag.String("latest", "2026-11-12", "the day of the newest reading (YYYY-MM-DD)")
	flag.Parse()

	day0, err := time.Parse(time.DateOnly, *latest)
	if err != nil {
		log.Fatalf("-latest: %v", err)
	}
	g := &generator{
		day0:    day0,
		rng:     rand.New(rand.NewPCG(855, 2026)),
		library: *library,
		media:   *media,
	}
	if err := g.run(context.Background(), *dbPath, *username); err != nil {
		log.Fatal(err)
	}
}

type generator struct {
	day0           time.Time
	rng            *rand.Rand
	library, media string

	tx     *sql.Tx
	userID int
	epubs  map[string]*epubBook

	readThroughs, sessions, updates int
}

type file struct {
	id, bookID int
	fileType   string
	path       string
	pages      int
	seconds    float64
}

// session is one stretch of reading, before it is written.
type session struct {
	start, end time.Time
	from, to   float64
}

// position is a place in a File, in the form the web reader saves for it.
type position struct {
	cfi        *string
	page       *int
	seconds    *float64
	percentage float64
}

func (g *generator) run(ctx context.Context, dbPath, username string) error {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	g.tx = tx
	g.epubs = map[string]*epubBook{}

	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE username = ?`, username).Scan(&g.userID); err != nil {
		return fmt.Errorf("finding user %q: %w", username, err)
	}
	for _, table := range []string{"reading_position_updates", "reading_sessions", "reading_positions", "read_throughs", "user_books"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE user_id = ?`, g.userID); err != nil {
			return fmt.Errorf("clearing %s: %w", table, err)
		}
	}
	// Number the new rows from where the remaining ones end, so a rerun
	// writes the same rows and the committed database only changes when the
	// plan does.
	for _, table := range []string{"reading_position_updates", "reading_sessions", "read_throughs"} {
		_, err := tx.ExecContext(ctx, `UPDATE sqlite_sequence SET seq = (SELECT COALESCE(MAX(id), 0) FROM `+table+`) WHERE name = ?`, table)
		if err != nil {
			return fmt.Errorf("renumbering %s: %w", table, err)
		}
	}

	// Each read's Sessions are drawn by day first, so reads that share a day
	// can be fitted around each other before anything is written.
	planned := make([][]session, len(plan))
	byDay := map[int][]*session{}
	files := make([]file, len(plan))
	for i, r := range plan {
		f, err := g.findFile(ctx, r)
		if err != nil {
			return err
		}
		files[i] = f
		if r.logged {
			continue
		}
		planned[i] = g.drawSessions(r, f)
		for j := range planned[i] {
			s := &planned[i][j]
			d := g.daysBefore(s.start)
			byDay[d] = append(byDay[d], s)
		}
	}
	for _, d := range slices.Sorted(maps.Keys(byDay)) {
		separate(byDay[d], g.rng)
	}
	for _, sessions := range planned {
		slices.SortFunc(sessions, func(a, b session) int { return a.start.Compare(b.start) })
	}

	ratings := map[int]rating{}
	for i, r := range plan {
		end, err := g.writeRead(ctx, r, files[i], planned[i])
		if err != nil {
			return fmt.Errorf("%s: %w", r.title, err)
		}
		if r.rating > 0 {
			ratings[files[i].bookID] = rating{units: r.rating, at: end.Add(3 * time.Minute)}
		}
	}
	for _, bookID := range slices.Sorted(maps.Keys(ratings)) {
		rt := ratings[bookID]
		_, err := tx.ExecContext(ctx, `INSERT INTO user_books (user_id, book_id, created_at, updated_at, rating) VALUES (?, ?, ?, ?, ?)`,
			g.userID, bookID, stamp(rt.at), stamp(rt.at), rt.units)
		if err != nil {
			return fmt.Errorf("rating book %d: %w", bookID, err)
		}
	}

	if err := g.dateLibrary(ctx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return err
	}
	log.Printf("wrote %d Read-throughs, %d Reading Sessions, %d position updates, and %d Ratings for %s, ending %s",
		g.readThroughs, g.sessions, g.updates, len(ratings), username, g.day0.Format(time.DateOnly))
	return nil
}

// dateLibrary dates each Book, and its Files, to when it was added to the
// library in the plan, an evening before anything in it was read.
func (g *generator) dateLibrary(ctx context.Context) error {
	rows, err := g.tx.QueryContext(ctx, `SELECT id, title FROM books`)
	if err != nil {
		return err
	}
	books := map[int]string{}
	for rows.Next() {
		var id int
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			rows.Close()
			return err
		}
		books[id] = title
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range slices.Sorted(maps.Keys(books)) {
		days, ok := added[books[id]]
		if !ok {
			days = firstImport
		}
		at := stamp(g.day(days).Add(19*time.Hour + time.Duration(id)*time.Minute))
		for _, q := range []string{
			`UPDATE books SET created_at = ? WHERE id = ?`,
			`UPDATE files SET created_at = ? WHERE book_id = ?`,
		} {
			if _, err := g.tx.ExecContext(ctx, q, at, id); err != nil {
				return fmt.Errorf("dating book %d: %w", id, err)
			}
		}
	}
	return nil
}

type rating struct {
	units int
	at    time.Time
}

func (g *generator) findFile(ctx context.Context, r read) (file, error) {
	var f file
	var pages sql.NullInt64
	var seconds sql.NullFloat64
	err := g.tx.QueryRowContext(ctx, `
		SELECT f.id, f.book_id, f.file_type, f.filepath, f.page_count, f.audiobook_duration_seconds
		FROM files AS f JOIN books AS b ON b.id = f.book_id
		WHERE b.title = ? AND f.file_type = ? AND f.file_role = 'main'`, r.title, r.fileType).
		Scan(&f.id, &f.bookID, &f.fileType, &f.path, &pages, &seconds)
	if errors.Is(err, sql.ErrNoRows) {
		return f, fmt.Errorf("no %s File of %q in the library", r.fileType, r.title)
	}
	if err != nil {
		return f, err
	}
	f.pages, f.seconds = int(pages.Int64), seconds.Float64
	return f, nil
}

func (g *generator) day(daysBefore int) time.Time {
	return g.day0.AddDate(0, 0, -daysBefore)
}

func (g *generator) daysBefore(t time.Time) int {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return int(math.Round(g.day0.Sub(day).Hours() / 24))
}

// drawSessions picks the stretches of reading for r, with the progress
// still to be filled in.
func (g *generator) drawSessions(r read, f file) []session {
	var sessions []session
	for d := r.from; d >= r.to; d-- {
		day := g.day(d)
		weekend := day.Weekday() == time.Saturday || day.Weekday() == time.Sunday
		must := d == r.from || d == r.to
		for _, w := range g.windows(r, f, weekend, must) {
			start := day.Add(w.at + g.jitter(w.spread))
			length := w.min + time.Duration(g.rng.Int64N(int64(w.max-w.min)+1))
			sessions = append(sessions, session{start: start.Truncate(time.Second), end: start.Add(length).Truncate(time.Second)})
		}
	}
	// A finished listen ends when the audio does, at about 1.1x, which can
	// be before the last day the plan allows.
	if f.seconds > 0 && r.outcome == finished {
		listened := 0.0
		for i, s := range sessions {
			listened += s.end.Sub(s.start).Seconds() * 1.1
			if listened >= f.seconds {
				sessions = sessions[:i+1]
				break
			}
		}
	}
	return sessions
}

// window is a time of day someone reads, and for how long.
type window struct {
	at, spread time.Duration
	min, max   time.Duration
}

func (g *generator) chance(p float64) bool { return g.rng.Float64() < p }

func (g *generator) jitter(spread time.Duration) time.Duration {
	if spread == 0 {
		return 0
	}
	return time.Duration(g.rng.Int64N(int64(2*spread))) - spread
}

// windows returns the Sessions r gets on one day. must is set on the first
// and last day, which always get one.
func (g *generator) windows(r read, f file, weekend, must bool) []window {
	h, m := time.Hour, time.Minute
	var ws []window
	switch r.pattern {
	case sitting:
		// A comic or a short audiobook, start to finish (or to giving up).
		length := 35 * m
		switch {
		case f.seconds > 0:
			length = time.Duration(f.seconds) * time.Second
		case f.pages > 0:
			length = time.Duration(f.pages) * 35 * time.Second
		}
		length = time.Duration(float64(length) * math.Max(r.reach, 0.15))
		at := 20*h + 30*m
		if weekend {
			at = 15 * h
		}
		ws = append(ws, window{at: at, spread: 90 * m, min: length * 9 / 10, max: length * 11 / 10})
	case evenings:
		if must || g.chance(0.7) {
			ws = append(ws, window{at: 21 * h, spread: 50 * m, min: 15 * m, max: 40 * m})
		}
		if weekend && g.chance(0.4) {
			ws = append(ws, window{at: 10*h + 30*m, spread: 60 * m, min: 20 * m, max: 60 * m})
		}
	case textbook:
		switch {
		case weekend && (must || g.chance(0.5)):
			ws = append(ws, window{at: 10 * h, spread: 45 * m, min: 30 * m, max: 60 * m})
		case !weekend && (must || g.chance(0.22)):
			ws = append(ws, window{at: 19 * h, spread: 20 * m, min: 20 * m, max: 40 * m})
		}
	case commute:
		switch {
		case weekend && (must || g.chance(0.5)):
			ws = append(ws, window{at: 11 * h, spread: 60 * m, min: 35 * m, max: 70 * m})
		case !weekend:
			if must || g.chance(0.85) {
				ws = append(ws, window{at: 7*h + 50*m, spread: 15 * m, min: 22 * m, max: 34 * m})
			}
			if g.chance(0.65) {
				ws = append(ws, window{at: 17*h + 35*m, spread: 15 * m, min: 22 * m, max: 34 * m})
			}
		}
	}
	return ws
}

// separate moves one day's Sessions apart where they overlap, keeping each
// one's length, so one person never reads two things at once.
func separate(day []*session, rng *rand.Rand) {
	slices.SortFunc(day, func(a, b *session) int { return a.start.Compare(b.start) })
	for i := 1; i < len(day); i++ {
		prev, s := day[i-1], day[i]
		gap := time.Duration(5+rng.IntN(25)) * time.Minute
		if s.start.Before(prev.end.Add(gap)) {
			length := s.end.Sub(s.start)
			s.start = prev.end.Add(gap)
			s.end = s.start.Add(length)
		}
	}
}

// writeRead writes r's Read-through and everything under it, and returns
// when it last changed.
func (g *generator) writeRead(ctx context.Context, r read, f file, sessions []session) (time.Time, error) {
	begin, end := g.day(r.from), g.day(r.to)
	var created, updated time.Time
	if len(sessions) > 0 {
		created, updated = sessions[0].start, sessions[len(sessions)-1].end
		if r.outcome == finished {
			// Finished on the day of the last Session.
			last := sessions[len(sessions)-1].start
			end = time.Date(last.Year(), last.Month(), last.Day(), 0, 0, 0, 0, time.UTC)
		}
	} else {
		// Logged by hand on the evening it was finished.
		created = end.Add(21*time.Hour + 40*time.Minute)
		updated = created
	}
	if r.outcome == abandoned {
		// Given up on a day or two after the last try.
		updated = updated.Add(time.Duration(20+g.rng.IntN(30)) * time.Hour)
		end = time.Date(updated.Year(), updated.Month(), updated.Day(), 0, 0, 0, 0, time.UTC)
		if g.daysBefore(end) < 0 {
			end, updated = g.day0, g.day0.Add(22*time.Hour)
		}
	}
	var endDay any
	if r.outcome != underway {
		endDay = end.Format(time.DateOnly)
	}

	res, err := g.tx.ExecContext(ctx, `
		INSERT INTO read_throughs (created_at, updated_at, user_id, book_id, begin_day, end_day, outcome, paused)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		stamp(created), stamp(updated), g.userID, f.bookID, begin.Format(time.DateOnly), endDay, r.outcome, r.paused)
	if err != nil {
		return updated, err
	}
	readThroughID, err := res.LastInsertId()
	if err != nil {
		return updated, err
	}
	g.readThroughs++
	if len(sessions) == 0 {
		return updated, nil
	}

	g.fillProgress(r, f, sessions)
	var first, last *position
	for i, s := range sessions {
		at, err := g.locate(f, s.to)
		if err != nil {
			return updated, err
		}
		if at != nil {
			s.to = at.percentage
		}
		if i > 0 {
			s.from = sessions[i-1].to
		}
		sessions[i] = s
		kind := "reading"
		if f.seconds > 0 {
			kind = "listening"
		}
		_, err = g.tx.ExecContext(ctx, `
			INSERT INTO reading_sessions (created_at, user_id, file_id, read_through_id, source, started_at, ended_at,
				start_percentage, end_percentage, measured, kind, duration)
			VALUES (?, ?, ?, ?, 'web_reader', ?, ?, ?, ?, TRUE, ?, ?)`,
			stamp(s.end), g.userID, f.id, readThroughID, stamp(s.start), stamp(s.end),
			s.from, s.to, kind, int(s.end.Sub(s.start).Seconds()))
		if err != nil {
			return updated, err
		}
		g.sessions++

		// The web reader logs where each Session left off. Shisho has no
		// locator of its own for MOBI and AZW3 here, so those Books keep
		// their Sessions without a place.
		if at == nil {
			continue
		}
		atEnd := r.outcome == finished && i == len(sessions)-1
		_, err = g.tx.ExecContext(ctx, `
			INSERT INTO reading_position_updates (created_at, user_id, file_id, read_through_id, cfi, page, seconds,
				percentage, source, occurred_at, at_end, auto_finished)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'web_reader', ?, ?, ?)`,
			stamp(s.end), g.userID, f.id, readThroughID, at.cfi, at.page, at.seconds, at.percentage, stamp(s.end), atEnd, atEnd)
		if err != nil {
			return updated, err
		}
		g.updates++
		if first == nil {
			first = at
		}
		last = at
	}
	if last != nil {
		firstAt, lastAt := sessions[0].end, sessions[len(sessions)-1].end
		_, err = g.tx.ExecContext(ctx, `
			INSERT INTO reading_positions (user_id, file_id, created_at, updated_at, cfi, page, seconds, percentage, source, occurred_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'web_reader', ?)
			ON CONFLICT (user_id, file_id) DO UPDATE SET updated_at = excluded.updated_at, cfi = excluded.cfi,
				page = excluded.page, seconds = excluded.seconds, percentage = excluded.percentage, occurred_at = excluded.occurred_at`,
			g.userID, f.id, stamp(firstAt), stamp(lastAt), last.cfi, last.page, last.seconds, last.percentage, stamp(lastAt))
		if err != nil {
			return updated, err
		}
	}
	return updated, nil
}

// fillProgress sets how far each Session got. Reading moves in proportion
// to time spent, give or take. A listen still underway moves with the audio
// at a speed of 1.1x, so a Session's progress matches its length, and a
// finished one spreads the audio over its Sessions.
func (g *generator) fillProgress(r read, f file, sessions []session) {
	if f.seconds > 0 && r.outcome == finished {
		// A finished listen spreads the whole book over the time spent.
		total := 0.0
		for _, s := range sessions {
			total += s.end.Sub(s.start).Seconds()
		}
		listened := 0.0
		for i, s := range sessions {
			sessions[i].from = listened / total
			listened += s.end.Sub(s.start).Seconds()
			sessions[i].to = listened / total
		}
		return
	}
	if f.seconds > 0 {
		done := 0.0
		for i := range sessions {
			listened := sessions[i].end.Sub(sessions[i].start).Seconds() * 1.1
			sessions[i].from = done
			done = math.Min(done+listened/f.seconds, 1)
			done = math.Min(done, 0.95)
			sessions[i].to = done
		}
		return
	}
	weights := make([]float64, len(sessions))
	total := 0.0
	for i, s := range sessions {
		weights[i] = s.end.Sub(s.start).Minutes() * (0.7 + 0.6*g.rng.Float64())
		total += weights[i]
	}
	done := 0.0
	for i := range sessions {
		sessions[i].from = done
		done += r.reach * weights[i] / total
		if i == len(sessions)-1 {
			done = r.reach
		}
		sessions[i].to = math.Min(done, 1)
	}
}

// locate returns the place fraction of the way through f, as the web reader
// saves it, or nil for a File Type it has no locator for here.
func (g *generator) locate(f file, fraction float64) (*position, error) {
	switch {
	case f.seconds > 0:
		seconds := math.Round(fraction*f.seconds*10) / 10
		return &position{seconds: &seconds, percentage: fraction}, nil
	case f.pages > 1:
		page := int(math.Round(fraction * float64(f.pages-1)))
		return &position{page: &page, percentage: float64(page) / float64(f.pages-1)}, nil
	case f.fileType == "epub":
		book, err := g.epub(f)
		if err != nil {
			return nil, err
		}
		cfi, at, err := book.locate(fraction)
		if err != nil {
			return nil, err
		}
		return &position{cfi: &cfi, percentage: at}, nil
	}
	return nil, nil
}

func (g *generator) epub(f file) (*epubBook, error) {
	if book, ok := g.epubs[f.path]; ok {
		return book, nil
	}
	rel, ok := strings.CutPrefix(f.path, g.media+"/")
	if !ok {
		return nil, fmt.Errorf("%s is outside %s", f.path, g.media)
	}
	book, err := openEPUB(filepath.Join(g.library, filepath.FromSlash(rel)))
	if err != nil {
		return nil, err
	}
	g.epubs[f.path] = book
	return book, nil
}

// stamp formats t the way Shisho stores instants, which its queries compare
// as text.
func stamp(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05.999999-07:00")
}

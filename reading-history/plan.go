package main

// The demo user's reading. Days count back from the latest day (0), so the
// newest reading lands on it. In Demo Mode, Shisho moves every date so that
// day 0 is yesterday, so where the year boundary falls depends on the day
// the demo is visited. Some of everything (finishes, a Did Not Finish,
// listening) sits in the last few months, so the current year has reading
// in it even early in the year.

// pattern is how a Read-through's Sessions fall across its days.
type pattern int

const (
	// evenings: most days, after dinner, and longer on weekends.
	evenings pattern = iota
	// sitting: the whole thing on its one day.
	sitting
	// commute: listening on weekday commutes and some weekend walks.
	commute
	// textbook: a couple of evenings a week and long weekend mornings.
	textbook
)

type read struct {
	title    string // the Book's title
	fileType string // the File read
	from, to int    // days before the latest day; equal for one sitting
	outcome  string
	// reach is how far it got: 1 for a finish. An audiobook's reach comes
	// from the time listened instead, so it is left 0 there.
	reach   float64
	paused  bool
	rating  int // quarter stars, 1 to 20; 0 is unrated
	pattern pattern
	// logged marks a Read-through entered by hand after the fact: days and
	// an outcome, with no Sessions.
	logged bool
}

const (
	finished  = "finished"
	abandoned = "abandoned"
	underway  = "underway"
)

var plan = []read{
	// Over a year ago: a read logged by hand, then the first ones tracked in
	// Shisho.
	{title: "A Christmas Carol", fileType: "epub", from: 412, to: 405, outcome: finished, reach: 1, rating: 18, logged: true},
	{title: "Planet Comics #1", fileType: "cbz", from: 398, to: 398, outcome: finished, reach: 1, rating: 12, pattern: sitting},
	{title: "Open Advice", fileType: "epub", from: 381, to: 346, outcome: finished, reach: 1, rating: 14, pattern: evenings},
	{title: "The Importance of Being Earnest", fileType: "epub", from: 341, to: 336, outcome: finished, reach: 1, rating: 18, pattern: evenings},

	// The past year.
	{title: "Frankenstein", fileType: "mobi", from: 318, to: 276, outcome: finished, reach: 1, rating: 18, pattern: evenings},
	{title: "Planet Comics #3", fileType: "cbz", from: 290, to: 290, outcome: finished, reach: 1, rating: 10, pattern: sitting},
	{title: "Made with Creative Commons", fileType: "pdf", from: 268, to: 251, outcome: abandoned, reach: 0.38, pattern: evenings},
	{title: "The Wonderful Wizard of Oz", fileType: "epub", from: 244, to: 205, outcome: finished, reach: 1, rating: 16, pattern: evenings},
	{title: "Software Engineering", fileType: "pdf", from: 224, to: 61, outcome: underway, reach: 0.52, paused: true, pattern: textbook},
	{title: "Alice’s Adventures in Wonderland", fileType: "azw3", from: 175, to: 150, outcome: finished, reach: 1, rating: 16, pattern: evenings},
	{title: "Planet Comics #5", fileType: "cbz", from: 100, to: 100, outcome: abandoned, reach: 0.22, rating: 8, pattern: sitting},
	{title: "The Velveteen Rabbit", fileType: "m4b", from: 75, to: 75, outcome: finished, reach: 1, rating: 20, pattern: sitting},
	{title: "The Unity Tree", fileType: "cbz", from: 64, to: 64, outcome: finished, reach: 1, rating: 18, pattern: sitting},
	{title: "There are no Shortcuts", fileType: "cbz", from: 63, to: 63, outcome: finished, reach: 1, rating: 18, pattern: sitting},
	{title: "A Christmas Carol", fileType: "epub", from: 52, to: 45, outcome: finished, reach: 1, rating: 18, pattern: evenings},
	{title: "Planet Comics #1", fileType: "cbz", from: 12, to: 12, outcome: finished, reach: 1, rating: 12, pattern: sitting},

	// Underway now: a novel in the evenings and a reread of Oz as an
	// audiobook on the way to work.
	{title: "The Marvelous Land of Oz", fileType: "epub", from: 31, to: 1, outcome: underway, reach: 0.58, pattern: evenings},
	{title: "The Wonderful Wizard of Oz", fileType: "m4b", from: 2, to: 0, outcome: underway, pattern: commute},
}

// added is how many days before the latest day each Book was added to
// the library, so Library Growth shows the library arriving ahead of the
// reading in it. A Book not listed came with the first import.
const firstImport = 440

var added = map[string]int{
	"Frankenstein":                     330,
	"Planet Comics #3":                 300,
	"Planet Comics #5":                 300,
	"Made with Creative Commons":       285,
	"Alice’s Adventures in Wonderland": 190,
	"The Unity Tree":                   120,
	"There are no Shortcuts":           120,
}

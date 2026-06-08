//go:build test

package database_test

import "fmt"

// seedDBDSN builds the DSN used by the seed-DB generator (TestGenerateSeedDB) and
// its regression test (TestSeededAccessLogSurvivesRollupOnRestart).
//
// It mirrors the production DSN's time-format params (NewSQLite's default branch):
// _time_format=sqlite, _texttotime=1, _timezone=UTC. These are correctness-critical
// — without _time_format, modernc.org/sqlite stores time.Time via Go's
// time.Time.String() format ("2006-01-02 15:04:05.999999999 +0000 UTC"), which
// SQLite's strftime cannot parse. The traffic-rollup then yields a NULL bucket_at,
// violating the NOT NULL constraint and crash-looping the app on restart (PW-68).
//
// journal_mode(DELETE) keeps the artifact a single self-contained file (no WAL
// sidecars). foreign_keys(1) matches production constraint enforcement.
func seedDBDSN(path string) string {
	return fmt.Sprintf(
		"file:%s?_time_format=sqlite&_texttotime=1&_timezone=UTC&_pragma=foreign_keys(1)&_pragma=journal_mode(DELETE)&_pragma=busy_timeout(5000)",
		path,
	)
}

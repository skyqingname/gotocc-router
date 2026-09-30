package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"testing/fstest"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/LuckyKuang/sub2api-plus/migrations"
	"github.com/stretchr/testify/require"
)

func TestLegacyMigrationLineageRulesMatchEmbeddedMigrations(t *testing.T) {
	legacyNames := make(map[string]struct{}, len(legacyMigrationLineageRules))
	targetNames := make(map[string]struct{}, len(legacyMigrationLineageRules))

	for _, rule := range legacyMigrationLineageRules {
		require.NotEmpty(t, rule.legacyFilename)
		require.NotEmpty(t, rule.targetFilename)
		require.Len(t, rule.legacyChecksum, 64)
		require.Len(t, rule.targetChecksum, 64)
		require.NotEmpty(t, rule.schemaCheckSQL)
		require.Equal(t, rule.legacyChecksum, rule.targetChecksum)

		_, duplicateLegacy := legacyNames[rule.legacyFilename]
		require.Falsef(t, duplicateLegacy, "duplicate legacy filename %s", rule.legacyFilename)
		legacyNames[rule.legacyFilename] = struct{}{}

		_, duplicateTarget := targetNames[rule.targetFilename]
		require.Falsef(t, duplicateTarget, "duplicate target filename %s", rule.targetFilename)
		targetNames[rule.targetFilename] = struct{}{}

		content, err := migrations.FS.ReadFile(rule.targetFilename)
		require.NoError(t, err)
		require.Equal(t, rule.targetChecksum, checksumMigrationContent(string(content)))
	}

	require.Len(t, legacyMigrationLineageRules, 14)
}

func TestPrepareLegacyMigrationLineageSkipsUnrelatedMigrationFS(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	err = prepareLegacyMigrationLineage(context.Background(), db, fstest.MapFS{
		"001_init.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrepareLegacyMigrationLineageRulesCleanDatabaseUsesNormalMigrations(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rule, fsys := lineageRuleFixture("legacy.sql", "target.sql", "SELECT TRUE")
	expectMigrationRecordMissing(mock, rule.legacyFilename)
	expectMigrationRecordMissing(mock, rule.targetFilename)

	err = prepareLegacyMigrationLineageRules(context.Background(), db, fsys, []legacyMigrationLineageRule{rule})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrepareLegacyMigrationLineageRulesRecordsReviewedEquivalent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rule, fsys := lineageRuleFixture("legacy.sql", "target.sql", "SELECT TRUE")
	expectMigrationRecord(mock, rule.legacyFilename, rule.legacyChecksum)
	expectMigrationRecordMissing(mock, rule.targetFilename)
	mock.ExpectQuery(regexp.QuoteMeta(rule.schemaCheckSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"matches"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)")).
		WithArgs(rule.targetFilename, rule.targetChecksum).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = prepareLegacyMigrationLineageRules(context.Background(), db, fsys, []legacyMigrationLineageRule{rule})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrepareLegacyMigrationLineageRulesIsIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rule, fsys := lineageRuleFixture("legacy.sql", "target.sql", "SELECT TRUE")
	expectMigrationRecord(mock, rule.legacyFilename, rule.legacyChecksum)
	expectMigrationRecord(mock, rule.targetFilename, rule.targetChecksum)
	mock.ExpectQuery(regexp.QuoteMeta(rule.schemaCheckSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"matches"}).AddRow(true))

	err = prepareLegacyMigrationLineageRules(context.Background(), db, fsys, []legacyMigrationLineageRule{rule})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrepareLegacyMigrationLineageRulesRejectsTargetFileChecksum(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rule, fsys := lineageRuleFixture("legacy.sql", "target.sql", "SELECT TRUE")
	rule.targetChecksum = checksumMigrationContent("different")

	err = prepareLegacyMigrationLineageRules(context.Background(), db, fsys, []legacyMigrationLineageRule{rule})
	require.Error(t, err)
	require.Contains(t, err.Error(), "target checksum mismatch")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrepareLegacyMigrationLineageRulesRejectsLegacyDatabaseChecksum(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rule, fsys := lineageRuleFixture("legacy.sql", "target.sql", "SELECT TRUE")
	expectMigrationRecord(mock, rule.legacyFilename, "unexpected")
	expectMigrationRecordMissing(mock, rule.targetFilename)

	err = prepareLegacyMigrationLineageRules(context.Background(), db, fsys, []legacyMigrationLineageRule{rule})
	require.Error(t, err)
	require.Contains(t, err.Error(), "legacy migration checksum mismatch")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrepareLegacyMigrationLineageRulesRejectsTargetDatabaseChecksum(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rule, fsys := lineageRuleFixture("legacy.sql", "target.sql", "SELECT TRUE")
	expectMigrationRecordMissing(mock, rule.legacyFilename)
	expectMigrationRecord(mock, rule.targetFilename, "unexpected")

	err = prepareLegacyMigrationLineageRules(context.Background(), db, fsys, []legacyMigrationLineageRule{rule})
	require.Error(t, err)
	require.Contains(t, err.Error(), "database checksum mismatch")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrepareLegacyMigrationLineageRulesRejectsSchemaMismatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rule, fsys := lineageRuleFixture("legacy.sql", "target.sql", "SELECT TRUE")
	expectMigrationRecord(mock, rule.legacyFilename, rule.legacyChecksum)
	expectMigrationRecordMissing(mock, rule.targetFilename)
	mock.ExpectQuery(regexp.QuoteMeta(rule.schemaCheckSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"matches"}).AddRow(false))

	err = prepareLegacyMigrationLineageRules(context.Background(), db, fsys, []legacyMigrationLineageRule{rule})
	require.Error(t, err)
	require.Contains(t, err.Error(), "schema mismatch")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrepareLegacyMigrationLineageRulesPreflightsAllBeforeTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	first, firstFS := lineageRuleFixture("legacy-1.sql", "target-1.sql", "SELECT TRUE")
	second, secondFS := lineageRuleFixture("legacy-2.sql", "target-2.sql", "SELECT FALSE")
	fsys := fstest.MapFS{
		first.targetFilename:  firstFS[first.targetFilename],
		second.targetFilename: secondFS[second.targetFilename],
	}

	expectMigrationRecord(mock, first.legacyFilename, first.legacyChecksum)
	expectMigrationRecordMissing(mock, first.targetFilename)
	mock.ExpectQuery(regexp.QuoteMeta(first.schemaCheckSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"matches"}).AddRow(true))
	expectMigrationRecord(mock, second.legacyFilename, second.legacyChecksum)
	expectMigrationRecordMissing(mock, second.targetFilename)
	mock.ExpectQuery(regexp.QuoteMeta(second.schemaCheckSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"matches"}).AddRow(false))

	err = prepareLegacyMigrationLineageRules(
		context.Background(),
		db,
		fsys,
		[]legacyMigrationLineageRule{first, second},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "schema mismatch")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrepareLegacyMigrationLineageRulesRollsBackRegistrationFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rule, fsys := lineageRuleFixture("legacy.sql", "target.sql", "SELECT TRUE")
	expectMigrationRecord(mock, rule.legacyFilename, rule.legacyChecksum)
	expectMigrationRecordMissing(mock, rule.targetFilename)
	mock.ExpectQuery(regexp.QuoteMeta(rule.schemaCheckSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"matches"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)")).
		WithArgs(rule.targetFilename, rule.targetChecksum).
		WillReturnError(errors.New("write failed"))
	mock.ExpectRollback()

	err = prepareLegacyMigrationLineageRules(context.Background(), db, fsys, []legacyMigrationLineageRule{rule})
	require.Error(t, err)
	require.Contains(t, err.Error(), "record equivalent migration")
	require.NoError(t, mock.ExpectationsWereMet())
}

func lineageRuleFixture(
	legacyFilename string,
	targetFilename string,
	schemaCheckSQL string,
) (legacyMigrationLineageRule, fstest.MapFS) {
	content := "SELECT 1;"
	checksum := checksumMigrationContent(content)
	rule := legacyMigrationLineageRule{
		legacyFilename: legacyFilename,
		targetFilename: targetFilename,
		legacyChecksum: checksum,
		targetChecksum: checksum,
		schemaCheckSQL: schemaCheckSQL,
	}
	return rule, fstest.MapFS{
		targetFilename: &fstest.MapFile{Data: []byte(content)},
	}
}

func expectMigrationRecord(mock sqlmock.Sqlmock, filename, checksum string) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT checksum FROM schema_migrations WHERE filename = $1")).
		WithArgs(filename).
		WillReturnRows(sqlmock.NewRows([]string{"checksum"}).AddRow(checksum))
}

func expectMigrationRecordMissing(mock sqlmock.Sqlmock, filename string) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT checksum FROM schema_migrations WHERE filename = $1")).
		WithArgs(filename).
		WillReturnRows(sqlmock.NewRows([]string{"checksum"}))
}

package e2ee

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSenderKeySchemaUses64BitVersionColumns(t *testing.T) {
	root := repoRoot(t)
	e2eeSQL := mustReadFile(t, filepath.Join(root, "dev-doc/sql/09_e2ee.sql"))

	assertCreateColumnIsBigint(t, e2eeSQL, "public.member_sender_keys", "chain_id")
	assertCreateColumnIsBigint(t, e2eeSQL, "public.member_sender_keys", "sender_key_version")
	assertCreateColumnIsBigint(t, e2eeSQL, "public.sender_key_distributions", "chain_id")
	assertCreateColumnIsBigint(t, e2eeSQL, "public.sender_key_distributions", "sender_key_version")
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)

	return filepath.Clean(filepath.Join(filepath.Dir(filename), "../../../../.."))
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}

func assertCreateColumnIsBigint(t *testing.T, sql, table, column string) {
	t.Helper()

	tablePattern := regexp.MustCompile(`(?is)CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+` + regexp.QuoteMeta(table) + `\s*\((.*?)\);`)
	match := tablePattern.FindStringSubmatch(sql)
	require.Len(t, match, 2, "missing create table block for %s", table)

	columnPattern := regexp.MustCompile(`(?im)^\s*` + regexp.QuoteMeta(column) + `\s+BIGINT\b`)
	require.True(t, columnPattern.MatchString(match[1]), "%s.%s must be BIGINT", table, column)
}

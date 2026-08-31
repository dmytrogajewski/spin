package nav

// Journey: specs/journeys/JOURNEY-022-structured-navigation-index.md.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/contexteng/compact"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/skills"
)

const (
	testSkillName   = "nav-probe"
	testSkillWhy    = "Probe skill for navigation index tests."
	testSkillDir    = "/tmp/nav-probe"
	testSkillSecret = "SECRET_SKILL_BODY_MUST_NOT_LEAK"
	testPluginName  = "valid-plugin"
	testSessionID   = "sess-nav-1"
	testSessionDir  = "/work/nav"
	testSessionBody = "SECRET_SESSION_TRANSCRIPT_MUST_NOT_LEAK"
	testPathSecret  = "SECRET_FILE_BODY_MUST_NOT_LEAK"
	testSymbolName  = "Lookup"
	testSymbolOpen  = "internal/nav/index.go:12"
	testSymbolBody  = "SECRET_SYMBOL_BODY_MUST_NOT_LEAK"
)

func TestRecords_SkillShape(t *testing.T) {
	t.Parallel()

	index := New(Sources{Skills: skills.Catalog{{
		Name:        testSkillName,
		Description: testSkillWhy,
		Location:    testSkillDir,
		Source:      skills.SourceProject,
	}}})

	records, err := index.Records(KindSkill)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, KindSkill, records[0].Kind)
	require.Equal(t, testSkillName, records[0].ID)
	require.Equal(t, testSkillName, records[0].Title)
	require.Equal(t, testSkillWhy, records[0].Why)
	require.Equal(t, testSkillDir, records[0].Open)
	assertRecordShape(t, records[0])
}

func TestRecords_SkillEscapesBody(t *testing.T) {
	t.Parallel()

	index := New(Sources{Skills: skills.Catalog{{
		Name:        testSkillName,
		Description: testSkillWhy,
		Location:    testSkillDir,
		Source:      skills.SourceProject,
	}}})

	records, err := index.Records(KindSkill)
	require.NoError(t, err)
	assertNoSecret(t, records, testSkillSecret)
	require.Equal(t, testSkillDir, records[0].Open)
}

func TestRecords_WhyOneLine(t *testing.T) {
	t.Parallel()

	index := New(Sources{Skills: skills.Catalog{{
		Name:        testSkillName,
		Description: "line one\nline two",
		Location:    testSkillDir,
		Source:      skills.SourceProject,
	}}})

	records, err := index.Records(KindSkill)
	require.NoError(t, err)
	require.NotContains(t, records[0].Why, "\n")
	require.Equal(t, "line one line two", records[0].Why)
}

func TestRecords_PluginFromCatalog(t *testing.T) {
	t.Parallel()

	const pluginRoot = "/plugins/valid-plugin"

	row := PluginRow{Name: testPluginName, Description: "Brief plugin description", Root: pluginRoot}
	records, recErr := New(Sources{Plugins: []PluginRow{row}}).Records(KindPlugin)
	require.NoError(t, recErr)
	require.Len(t, records, 1)
	require.Equal(t, KindPlugin, records[0].Kind)
	require.Equal(t, testPluginName, records[0].ID)
	require.Equal(t, pluginRoot, records[0].Open)
	assertRecordShape(t, records[0])
	assertNoSecret(t, records, `"$schema"`)
}

func TestRecords_SessionFromResumeIndex(t *testing.T) {
	t.Parallel()

	resume := newTestResume(t)
	require.NoError(t, resume.Update(t.Context(), session.IndexEntry{
		ID:      testSessionID,
		Title:   "nav session",
		WorkDir: testSessionDir,
	}))

	records, err := New(Sources{Sessions: resume}).Records(KindSession)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, KindSession, records[0].Kind)
	require.Equal(t, testSessionID, records[0].ID)
	require.Equal(t, testSessionDir, records[0].Open)
	assertRecordShape(t, records[0])
	assertNoSecret(t, records, testSessionBody)
}

func TestRecords_PeersFromBuiltins(t *testing.T) {
	t.Parallel()

	records, err := New(Sources{}).Records(KindPeer)
	require.NoError(t, err)
	require.Len(t, records, len(subagent.Builtins()))

	names := make(map[string]Record, len(records))
	for _, record := range records {
		assertRecordShape(t, record)
		require.Equal(t, KindPeer, record.Kind)
		require.True(t, strings.HasPrefix(record.Open, "stdio://"))
		names[record.ID] = record
	}

	for _, spec := range subagent.Builtins() {
		_, ok := names[spec.Name]
		require.True(t, ok, "missing peer %s", spec.Name)
	}
}

func TestRecords_UnknownKind(t *testing.T) {
	t.Parallel()

	_, err := New(Sources{}).Records(Kind("nope"))
	require.ErrorIs(t, err, ErrUnknownKind)
	require.Contains(t, err.Error(), ValidKinds)
}

func TestPaths_TreeCompressed(t *testing.T) {
	t.Parallel()

	dir, raw := writePathFixture(t)
	result, err := New(Sources{}).Paths(dir)
	require.NoError(t, err)

	want := string(compact.Default().Apply("ls", []byte(raw), nil, 0).Stdout)
	require.Equal(t, want, result.Listing)
	require.NotEqual(t, raw, result.Listing)
	require.Contains(t, result.Listing, ". (")
	require.Len(t, result.Records, 1)
	require.Equal(t, KindPath, result.Records[0].Kind)
	require.Equal(t, dir, result.Records[0].Open)
	assertRecordShape(t, result.Records[0])
	assertNoSecret(t, result.Records, testPathSecret)
	require.NotContains(t, result.Listing, testPathSecret)
}

func TestRecords_SymbolPointer(t *testing.T) {
	t.Parallel()

	index := New(Sources{Symbols: symbolList{{
		Name: testSymbolName,
		Open: testSymbolOpen,
		Why:  "func",
	}}})

	records, err := index.Records(KindSymbol)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, KindSymbol, records[0].Kind)
	require.Equal(t, testSymbolOpen, records[0].Open)
	assertRecordShape(t, records[0])
	assertNoSecret(t, records, testSymbolBody)
}

func TestDiscover_LivePluginCatalog(t *testing.T) {
	t.Parallel()

	const pluginRoot = "/plugins/valid-plugin"

	index := Discover(DiscoverOpts{
		HomeDir: t.TempDir(),
		Plugins: []PluginRow{{
			Name:        testPluginName,
			Description: "Brief plugin description",
			Root:        pluginRoot,
		}},
	})
	records, err := index.Records(KindPlugin)
	require.NoError(t, err)
	require.NotEmpty(t, records)
	require.Equal(t, testPluginName, records[0].ID)
	require.Equal(t, pluginRoot, records[0].Open)
}

func TestDiscover_LiveSkillCatalog(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	skillDir := filepath.Join(workDir, ".agents", "skills", testSkillName)
	require.NoError(t, os.MkdirAll(skillDir, 0o750))

	body := "---\nname: " + testSkillName + "\ndescription: " + testSkillWhy + "\n---\n\n" + testSkillSecret + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, skills.FileName), []byte(body), 0o600))

	index := Discover(DiscoverOpts{WorkDir: workDir, HomeDir: t.TempDir()})
	records, err := index.Records(KindSkill)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, testSkillName, records[0].ID)
	require.Equal(t, skillDir, records[0].Open)
	assertNoSecret(t, records, testSkillSecret)
}

func TestLookup_PathAndFilter(t *testing.T) {
	t.Parallel()

	index := New(Sources{Skills: skills.Catalog{
		{Name: testSkillName, Description: testSkillWhy, Location: testSkillDir},
		{Name: "other-skill", Description: "other", Location: "/tmp/other"},
	}})

	result, err := index.Lookup(Query{Kind: KindSkill, ID: testSkillName})
	require.NoError(t, err)
	require.Len(t, result.Records, 1)
	require.Equal(t, testSkillName, result.Records[0].ID)
}

type symbolList []SymbolHit

func (list symbolList) Find(string) []SymbolHit {
	return list
}

func newTestResume(t *testing.T) *session.Index {
	t.Helper()

	idx, err := session.NewSessionIndex(t.Context(), filepath.Join(t.TempDir(), "index.json"), nil)
	require.NoError(t, err)

	return idx
}

func writePathFixture(t *testing.T) (dir, rawListing string) {
	t.Helper()

	dir = t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file1.txt"), []byte(testPathSecret), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("b"), 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir"), 0o750))

	return dir, "./file1.txt\n./file2.txt\n./subdir/\n"
}

func assertRecordShape(t *testing.T, record Record) {
	t.Helper()
	require.NotEmpty(t, record.Kind)
	require.NotEmpty(t, record.ID)
	require.NotEmpty(t, record.Title)
	require.NotEmpty(t, record.Why)
	require.NotEmpty(t, record.Open)
}

func assertNoSecret(t *testing.T, records []Record, secret string) {
	t.Helper()

	payload, err := json.Marshal(records)
	require.NoError(t, err)
	require.NotContains(t, string(payload), secret)
}

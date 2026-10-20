package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// legacyShareDeck is the bundled sharesnap.txt from before the low-mention
// deck. Its sha256 is pinned in api_assets.go; the upgrade only touches a file
// that still matches it.
const legacyShareDeck = `# API Chat snap share deck. One line = one share message.
# %username% is replaced by the app with a dashboard handle. Never write @ usernames here.
ok fine, add me on snap %username%
my snap is %username%, add me rn
sc: %username% .. come find me there
ok u got me lol, snap %username%
just add me on snap, %username%
snap me: %username%
u on snap? its %username%
add my snap %username% before i change my mind
fine, %username% .. dont make it weird
my sc is %username%, u better add me
okok %username%
%username% .. thats it, thats the snap
find me on snap: %username%
snap: %username% (dont lose it)
u got lucky, %username%
add %username% and say hi
sc %username%, i barely check this app
ok here: %username%
`

func readDeck(t *testing.T, dataDir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dataDir, "api_chat", "sharesnap.txt"))
	if err != nil {
		t.Fatalf("read sharesnap.txt: %v", err)
	}
	return string(data)
}

func TestShareDeckNamesTheAppRarely(t *testing.T) {
	data, err := apiChatAssets.ReadFile("api_assets/sharesnap.txt")
	if err != nil {
		t.Fatalf("read embedded deck: %v", err)
	}
	lines := 0
	named := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines++
		if !strings.Contains(line, "%username%") {
			t.Errorf("share line without username placeholder: %q", line)
		}
		if strings.Contains(line, "@") {
			t.Errorf("share line has an @ handle: %q", line)
		}
		low := strings.ToLower(line)
		if strings.Contains(low, "snap") || containsWord(low, "sc") || containsWord(low, "snp") {
			named++
		}
	}
	if lines < 15 {
		t.Fatalf("share deck too small: %d lines", lines)
	}
	if named*100/lines > 30 {
		t.Fatalf("too many share lines name the app: %d/%d", named, lines)
	}
}

func containsWord(line, word string) bool {
	for _, field := range strings.FieldsFunc(line, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		if field == word {
			return true
		}
	}
	return false
}

func TestEnsureAPIChatDataUpgradesTheUntouchedOldDeck(t *testing.T) {
	dir := t.TempDir()
	if _, err := EnsureAPIChatData(dir); err != nil {
		t.Fatalf("seed: %v", err)
	}
	path := filepath.Join(dir, "api_chat", "sharesnap.txt")
	if err := os.WriteFile(path, []byte(legacyShareDeck), 0o644); err != nil {
		t.Fatalf("write legacy: %v", err)
	}
	if _, err := EnsureAPIChatData(dir); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if got := readDeck(t, dir); got == legacyShareDeck {
		t.Fatal("the untouched old deck must be upgraded")
	} else if !strings.Contains(got, "ok fine %username%") {
		t.Fatalf("unexpected upgraded deck: %q", got)
	}
}

func TestEnsureAPIChatDataKeepsAnEditedDeck(t *testing.T) {
	dir := t.TempDir()
	if _, err := EnsureAPIChatData(dir); err != nil {
		t.Fatalf("seed: %v", err)
	}
	path := filepath.Join(dir, "api_chat", "sharesnap.txt")
	custom := "# mine\nmy own line %username%\n"
	if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
		t.Fatalf("write custom: %v", err)
	}
	if _, err := EnsureAPIChatData(dir); err != nil {
		t.Fatalf("reseed: %v", err)
	}
	if got := readDeck(t, dir); got != custom {
		t.Fatalf("an edited deck was overwritten: %q", got)
	}
}

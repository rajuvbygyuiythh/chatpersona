package bot

// API Chat ships with a small editable data deck (persona + share + fallback).
// Files are seeded into data/api_chat only when missing, so a user's edits are
// never overwritten by an app update.

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed api_assets/README.txt api_assets/persona.txt api_assets/persona_countries.txt api_assets/sharesnap.txt api_assets/share.txt api_assets/fallback.txt
var apiChatAssets embed.FS

// EnsureAPIChatData creates data/api_chat and restores any missing starter
// file. It returns the runtime folder path.
func EnsureAPIChatData(dataDir string) (string, error) {
	root := filepath.Join(dataDir, "api_chat")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create API Chat data directory: %w", err)
	}
	upgradeShareDeck(root)
	err := fs.WalkDir(apiChatAssets, "api_assets", func(assetPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if assetPath == "api_assets" || entry.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(assetPath, "api_assets/")
		destination := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(destination); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
		data, err := apiChatAssets.ReadFile(assetPath)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0o644)
	})
	if err != nil {
		return "", fmt.Errorf("bootstrap API Chat starter data: %w", err)
	}
	return root, nil
}

// legacyShareDeckSHA256 is the exact bundled sharesnap.txt from before the
// low-mention deck. Only a file that still matches it is replaced: any edit by
// the owner changes the hash, so personal decks are never touched.
const legacyShareDeckSHA256 = "2f414450945c0e0640cf96b98e1eaa898dc71afbc23d44026d42941c4e5d2d2b"

// upgradeShareDeck swaps the untouched v1 deck for the new one where most
// lines do not name the app at all. It is silent on any error: a read-only or
// edited file simply keeps working.
func upgradeShareDeck(root string) {
	path := filepath.Join(root, "sharesnap.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	// Windows editors may have saved the old deck with CRLF: compare the same
	// bytes the embed uses, so an untouched copy still upgrades.
	normalized := []byte(strings.ReplaceAll(string(data), "\r\n", "\n"))
	sum := sha256.Sum256(normalized)
	if hex.EncodeToString(sum[:]) != legacyShareDeckSHA256 {
		return
	}
	fresh, err := apiChatAssets.ReadFile("api_assets/sharesnap.txt")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, fresh, 0o644)
}

// readAPILines reads a small TXT deck: one entry per line, blank lines and
// lines starting with # are ignored.
func readAPILines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

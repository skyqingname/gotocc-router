package service

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	archivepath "path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/releasechannel"
)

var (
	ErrNoUpdateAvailable         = infraerrors.Conflict("ALREADY_UP_TO_DATE", "no update available; current version is latest")
	ErrRollbackVersionNotAllowed = infraerrors.BadRequest("ROLLBACK_VERSION_NOT_ALLOWED", "version is not in the allowed rollback list")
)

const (
	updateCacheTTL       = 1200 // 20 minutes
	githubRepo           = releasechannel.ReleaseRepository
	upstreamRepo         = releasechannel.UpstreamRepository
	upstreamCheckTimeout = 5 * time.Second

	// Security: allowed download domains for updates
	allowedDownloadHost = "github.com"
	allowedAssetHost    = "objects.githubusercontent.com"

	// Security: max download size (500MB)
	maxDownloadSize = 500 * 1024 * 1024

	// Rollback: expose at most the 3 most recent versions older than current
	maxRollbackVersions = 3
	// Fetch a few extra releases so filtering (current/newer/prerelease) still leaves enough candidates
	rollbackFetchPageSize = 15
)

// UpdateCache defines cache operations for update service
type UpdateCache interface {
	GetUpdateInfo(ctx context.Context) (string, error)
	SetUpdateInfo(ctx context.Context, data string, ttl time.Duration) error
}

// GitHubReleaseClient 获取 GitHub release 信息的接口
type GitHubReleaseClient interface {
	FetchLatestRelease(ctx context.Context, repo string) (*GitHubRelease, error)
	FetchRecentReleases(ctx context.Context, repo string, perPage int) ([]*GitHubRelease, error)
	DownloadFile(ctx context.Context, url, dest string, maxSize int64) error
	FetchChecksumFile(ctx context.Context, url string) ([]byte, error)
}

// UpdateService handles software updates
type UpdateService struct {
	cache          UpdateCache
	githubClient   GitHubReleaseClient
	currentVersion string
	buildType      string // "source" for manual builds, "release" for CI builds
}

// NewUpdateService creates a new UpdateService
func NewUpdateService(cache UpdateCache, githubClient GitHubReleaseClient, version, buildType string) *UpdateService {
	return &UpdateService{
		cache:          cache,
		githubClient:   githubClient,
		currentVersion: version,
		buildType:      buildType,
	}
}

// UpdateInfo contains update information
type UpdateInfo struct {
	CurrentVersion        string       `json:"current_version"`
	LatestVersion         string       `json:"latest_version"`
	HasUpdate             bool         `json:"has_update"`
	ReleaseInfo           *ReleaseInfo `json:"release_info,omitempty"`
	Cached                bool         `json:"cached"`
	Warning               string       `json:"warning,omitempty"`
	BuildType             string       `json:"build_type"` // "source" or "release"
	ReleaseRepository     string       `json:"release_repository"`
	ReleaseImage          string       `json:"release_image"`
	UpstreamRepository    string       `json:"upstream_repository"`
	UpstreamBaseline      string       `json:"upstream_baseline"`
	UpstreamLatestVersion string       `json:"upstream_latest_version"`
	UpstreamHasUpdate     bool         `json:"upstream_has_update"`
	UpstreamWarning       string       `json:"upstream_warning,omitempty"`
}

// ReleaseInfo contains GitHub release details
type ReleaseInfo struct {
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	PublishedAt string  `json:"published_at"`
	HTMLURL     string  `json:"html_url"`
	Assets      []Asset `json:"assets,omitempty"`
}

// Asset represents a release asset
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
	Size        int64  `json:"size"`
}

// GitHubRelease represents GitHub API response
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	PublishedAt string        `json:"published_at"`
	HTMLURL     string        `json:"html_url"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	Assets      []GitHubAsset `json:"assets"`
}

// RollbackVersion describes a release version the system can roll back to
type RollbackVersion struct {
	Version     string `json:"version"` // without "v" prefix, e.g. "0.1.146"
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckUpdate checks for available updates
func (s *UpdateService) CheckUpdate(ctx context.Context, force bool) (*UpdateInfo, error) {
	// Try cache first
	if !force {
		if cached, err := s.getFromCache(ctx); err == nil && cached != nil {
			return cached, nil
		}
	}

	// Fetch from GitHub
	info, err := s.fetchLatestRelease(ctx)
	if err != nil {
		// Return cached on error
		if cached, cacheErr := s.getFromCache(ctx); cacheErr == nil && cached != nil {
			cached.Warning = "Using cached data: " + err.Error()
			return cached, nil
		}
		return &UpdateInfo{
			CurrentVersion:        s.currentVersion,
			LatestVersion:         s.currentVersion,
			HasUpdate:             false,
			Warning:               err.Error(),
			BuildType:             s.buildType,
			ReleaseRepository:     githubRepo,
			ReleaseImage:          releasechannel.ReleaseImage,
			UpstreamRepository:    upstreamRepo,
			UpstreamBaseline:      strings.TrimPrefix(releasechannel.UpstreamBaseline, "v"),
			UpstreamLatestVersion: strings.TrimPrefix(releasechannel.UpstreamBaseline, "v"),
		}, nil
	}

	// Cache result
	s.saveToCache(ctx, info)
	return info, nil
}

// PerformUpdate downloads and applies the update
// Uses atomic file replacement pattern for safe in-place updates
func (s *UpdateService) PerformUpdate(ctx context.Context) error {
	// An update must use fresh release metadata. Unlike the version display,
	// do not turn a GitHub or proxy failure into an "already up to date" result.
	info, err := s.fetchLatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("failed to check latest release: %w", err)
	}

	if !info.HasUpdate {
		return ErrNoUpdateAvailable
	}
	if info.ReleaseInfo == nil {
		return fmt.Errorf("update release metadata is missing")
	}

	return s.applyReleaseAssets(ctx, info.LatestVersion, info.ReleaseInfo.Assets)
}

// applyReleaseAssets downloads the platform archive from the given release assets,
// verifies its checksum, and installs the binary with its matched runtime files.
// Shared by PerformUpdate (latest) and RollbackToVersion (specific older version).
func (s *UpdateService) applyReleaseAssets(ctx context.Context, version string, releaseAssets []Asset) error {
	// Find matching archive and checksum for current platform
	archiveName := s.getArchiveName(version)
	var downloadURL string
	var checksumURL string

	for _, asset := range releaseAssets {
		if asset.Name == archiveName {
			downloadURL = asset.DownloadURL
		}
		if asset.Name == "checksums.txt" {
			checksumURL = asset.DownloadURL
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no compatible release found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// SECURITY: Validate download URL is from trusted domain
	if err := validateDownloadURL(downloadURL); err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	if checksumURL != "" {
		if err := validateDownloadURL(checksumURL); err != nil {
			return fmt.Errorf("invalid checksum URL: %w", err)
		}
	}

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	exeDir := filepath.Dir(exePath)

	// Create temp directory in the SAME directory as executable
	// This ensures os.Rename is atomic (same filesystem)
	tempDir, err := os.MkdirTemp(exeDir, ".sub2api-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	archivePath, err := s.downloadAndVerifyArchive(ctx, tempDir, archiveName, downloadURL, checksumURL)
	if err != nil {
		return err
	}

	stagedFiles, err := s.extractReleaseFiles(archivePath, tempDir, exePath)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	return installReleaseFiles(stagedFiles)
}

type stagedReleaseFile struct {
	archivePath string
	stagedPath  string
	targetPath  string
	mode        os.FileMode
}

type installedReleaseFile struct {
	targetPath  string
	backupPath  string
	hadOriginal bool
}

type rollbackReleaseFile struct {
	targetPath string
	backupPath string
	currentTmp string
	hadCurrent bool
}

// Rollback restores the previous matched release set.
func (s *UpdateService) Rollback() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	if _, err := os.Stat(exePath + ".backup"); os.IsNotExist(err) {
		return fmt.Errorf("no backup found")
	}

	targets := []string{exePath}
	exeDir := filepath.Dir(exePath)
	for _, runtimeFile := range releasechannel.RuntimeFiles {
		targets = append(targets, filepath.Join(exeDir, filepath.FromSlash(runtimeFile.Path)))
	}
	return restoreReleaseBackups(targets)
}

// ListRollbackVersions returns up to maxRollbackVersions release versions that are
// strictly older than the current version (the current version itself is excluded),
// newest first. Draft and prerelease entries are skipped.
func (s *UpdateService) ListRollbackVersions(ctx context.Context) ([]RollbackVersion, error) {
	releases, err := s.fetchRollbackCandidates(ctx)
	if err != nil {
		return nil, err
	}

	versions := make([]RollbackVersion, 0, len(releases))
	for _, r := range releases {
		versions = append(versions, RollbackVersion{
			Version:     strings.TrimPrefix(r.TagName, "v"),
			PublishedAt: r.PublishedAt,
			HTMLURL:     r.HTMLURL,
		})
	}
	return versions, nil
}

// RollbackToVersion downloads and installs a specific older version.
// The target must be one of the versions returned by ListRollbackVersions;
// anything else (including the current version) is rejected.
func (s *UpdateService) RollbackToVersion(ctx context.Context, version string) error {
	target := strings.TrimPrefix(strings.TrimSpace(version), "v")
	if target == "" {
		return ErrRollbackVersionNotAllowed
	}

	releases, err := s.fetchRollbackCandidates(ctx)
	if err != nil {
		return err
	}

	var match *GitHubRelease
	for _, r := range releases {
		if strings.TrimPrefix(r.TagName, "v") == target {
			match = r
			break
		}
	}
	if match == nil {
		return ErrRollbackVersionNotAllowed
	}

	assets := make([]Asset, len(match.Assets))
	for i, a := range match.Assets {
		assets[i] = Asset{
			Name:        a.Name,
			DownloadURL: a.BrowserDownloadURL,
			Size:        a.Size,
		}
	}

	return s.applyReleaseAssets(ctx, match.TagName, assets)
}

// fetchRollbackCandidates fetches recent releases and keeps the newest
// maxRollbackVersions entries strictly older than the current version.
func (s *UpdateService) fetchRollbackCandidates(ctx context.Context) ([]*GitHubRelease, error) {
	releases, err := s.githubClient.FetchRecentReleases(ctx, githubRepo, rollbackFetchPageSize)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(releases))
	candidates := make([]*GitHubRelease, 0, maxRollbackVersions)
	for _, r := range releases {
		if r == nil || r.Draft || r.Prerelease {
			continue
		}
		v := strings.TrimPrefix(r.TagName, "v")
		if v == "" || seen[v] {
			continue
		}
		if _, ok := parseForkReleaseVersion(v); !ok {
			continue
		}
		// Only versions strictly older than current (also excludes current itself)
		if compareVersions(v, s.currentVersion) >= 0 {
			continue
		}
		seen[v] = true
		candidates = append(candidates, r)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return compareVersions(
			strings.TrimPrefix(candidates[i].TagName, "v"),
			strings.TrimPrefix(candidates[j].TagName, "v"),
		) > 0
	})

	if len(candidates) > maxRollbackVersions {
		candidates = candidates[:maxRollbackVersions]
	}
	return candidates, nil
}

func (s *UpdateService) fetchLatestRelease(ctx context.Context) (*UpdateInfo, error) {
	release, err := s.githubClient.FetchLatestRelease(ctx, githubRepo)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, fmt.Errorf("GitHub returned an empty latest release")
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	if _, ok := parseForkReleaseVersion(latestVersion); !ok {
		return nil, fmt.Errorf("latest release tag %q is not a valid fork version (expected vX.Y.Z+custom.NNN)", release.TagName)
	}

	assets := make([]Asset, len(release.Assets))
	for i, a := range release.Assets {
		assets[i] = Asset{
			Name:        a.Name,
			DownloadURL: a.BrowserDownloadURL,
			Size:        a.Size,
		}
	}

	upstreamBaseline := strings.TrimPrefix(releasechannel.UpstreamBaseline, "v")
	upstreamLatest := upstreamBaseline
	upstreamWarning := ""
	upstreamCtx, cancelUpstream := context.WithTimeout(ctx, upstreamCheckTimeout)
	defer cancelUpstream()
	upstreamRelease, upstreamErr := s.githubClient.FetchLatestRelease(upstreamCtx, upstreamRepo)
	if upstreamErr != nil {
		upstreamWarning = upstreamErr.Error()
	} else if upstreamRelease == nil {
		upstreamWarning = "GitHub returned an empty upstream latest release"
	} else {
		candidate := strings.TrimPrefix(upstreamRelease.TagName, "v")
		if _, ok := parseForkReleaseVersion(candidate); !ok {
			upstreamWarning = fmt.Sprintf("upstream latest release tag %q is not a valid fork version", upstreamRelease.TagName)
		} else {
			upstreamLatest = candidate
		}
	}

	return &UpdateInfo{
		CurrentVersion: s.currentVersion,
		LatestVersion:  latestVersion,
		HasUpdate:      compareVersions(s.currentVersion, latestVersion) < 0,
		ReleaseInfo: &ReleaseInfo{
			Name:        release.Name,
			Body:        release.Body,
			PublishedAt: release.PublishedAt,
			HTMLURL:     release.HTMLURL,
			Assets:      assets,
		},
		Cached:                false,
		BuildType:             s.buildType,
		ReleaseRepository:     githubRepo,
		ReleaseImage:          releasechannel.ReleaseImage,
		UpstreamRepository:    upstreamRepo,
		UpstreamBaseline:      upstreamBaseline,
		UpstreamLatestVersion: upstreamLatest,
		UpstreamHasUpdate:     compareVersions(upstreamBaseline, upstreamLatest) < 0,
		UpstreamWarning:       upstreamWarning,
	}, nil
}

func (s *UpdateService) downloadFile(ctx context.Context, downloadURL, dest string) error {
	return s.githubClient.DownloadFile(ctx, downloadURL, dest, maxDownloadSize)
}

func (s *UpdateService) downloadAndVerifyArchive(ctx context.Context, tempDir, archiveName, downloadURL, checksumURL string) (string, error) {
	// GitHub encodes '+' in browser download URLs as '%2B'. Checksums use the
	// release asset name, so the local filename must come from archiveName.
	archivePath := filepath.Join(tempDir, archiveName)
	if err := s.downloadFile(ctx, downloadURL, archivePath); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	if checksumURL != "" {
		if err := s.verifyChecksum(ctx, archivePath, checksumURL); err != nil {
			return "", fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	return archivePath, nil
}

func (s *UpdateService) getArchiveName(version string) string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	return fmt.Sprintf("sub2api_%s_%s_%s.tar.gz", strings.TrimPrefix(version, "v"), osName, arch)
}

// validateDownloadURL checks if the URL is from an allowed domain
// SECURITY: This prevents SSRF and ensures downloads only come from trusted GitHub domains
func validateDownloadURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Must be HTTPS
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed")
	}

	// Check against allowed hosts
	host := parsedURL.Host
	// GitHub release URLs can be from github.com or objects.githubusercontent.com
	if host != allowedDownloadHost &&
		!strings.HasSuffix(host, "."+allowedDownloadHost) &&
		host != allowedAssetHost &&
		!strings.HasSuffix(host, "."+allowedAssetHost) {
		return fmt.Errorf("download from untrusted host: %s", host)
	}

	return nil
}

func (s *UpdateService) verifyChecksum(ctx context.Context, filePath, checksumURL string) error {
	// Download checksums file
	checksumData, err := s.githubClient.FetchChecksumFile(ctx, checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}

	// Calculate file hash
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	// Find expected hash in checksums file
	fileName := filepath.Base(filePath)
	scanner := bufio.NewScanner(strings.NewReader(string(checksumData)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == fileName {
			if parts[0] == actualHash {
				return nil
			}
			return fmt.Errorf("checksum mismatch: expected %s, got %s", parts[0], actualHash)
		}
	}

	return fmt.Errorf("checksum not found for %s", fileName)
}

func (s *UpdateService) extractReleaseFiles(archivePath, tempDir, exePath string) ([]stagedReleaseFile, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var reader io.Reader = f

	if strings.HasSuffix(archivePath, ".gz") {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer func() { _ = gzr.Close() }()
		reader = gzr
	}

	if !strings.Contains(archivePath, ".tar") {
		return nil, fmt.Errorf("unsupported release archive: %s", filepath.Base(archivePath))
	}

	desired := make(map[string]releasechannel.RuntimeFile, len(releasechannel.RuntimeFiles))
	for _, runtimeFile := range releasechannel.RuntimeFiles {
		desired[runtimeFile.Path] = runtimeFile
	}
	found := make(map[string]stagedReleaseFile, len(desired)+1)
	stageRoot := filepath.Join(tempDir, "release")
	tr := tar.NewReader(reader)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		name := archivepath.Clean(strings.TrimPrefix(filepath.ToSlash(hdr.Name), "./"))
		if name == "." || archivepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
			return nil, fmt.Errorf("path traversal attempt detected: %s", hdr.Name)
		}

		archiveName := ""
		mode := os.FileMode(0o644)
		if archivepath.Base(name) == "sub2api" || archivepath.Base(name) == "sub2api.exe" {
			archiveName = "sub2api"
			mode = 0o755
		} else if _, ok := desired[name]; ok {
			archiveName = name
		}
		if archiveName == "" {
			continue
		}
		if _, exists := found[archiveName]; exists {
			return nil, fmt.Errorf("release archive repeats %s", archiveName)
		}
		if hdr.Size < 0 || hdr.Size > maxDownloadSize {
			return nil, fmt.Errorf("release file %s has invalid size %d", archiveName, hdr.Size)
		}

		stagedPath := filepath.Join(stageRoot, filepath.FromSlash(archiveName))
		if err := os.MkdirAll(filepath.Dir(stagedPath), 0o755); err != nil {
			return nil, err
		}
		out, err := os.OpenFile(stagedPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return nil, err
		}
		written, copyErr := io.Copy(out, io.LimitReader(tr, hdr.Size+1))
		closeErr := out.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if written != hdr.Size {
			return nil, fmt.Errorf("release file %s size mismatch", archiveName)
		}

		targetPath := exePath
		if archiveName != "sub2api" {
			targetPath = filepath.Join(filepath.Dir(exePath), filepath.FromSlash(archiveName))
		}
		found[archiveName] = stagedReleaseFile{
			archivePath: archiveName,
			stagedPath:  stagedPath,
			targetPath:  targetPath,
			mode:        mode,
		}
	}

	binary, ok := found["sub2api"]
	if !ok {
		return nil, fmt.Errorf("binary not found in archive")
	}
	files := []stagedReleaseFile{binary}
	for _, runtimeFile := range releasechannel.RuntimeFiles {
		staged, ok := found[runtimeFile.Path]
		if !ok {
			if runtimeFile.Required {
				return nil, fmt.Errorf("required runtime file not found in archive: %s", runtimeFile.Path)
			}
			continue
		}
		files = append(files, staged)
	}
	return files, nil
}

func installReleaseFiles(files []stagedReleaseFile) error {
	installed := make([]installedReleaseFile, 0, len(files))
	for _, file := range files {
		if err := os.Chmod(file.stagedPath, file.mode); err != nil {
			return restoreInstalledFiles(installed, fmt.Errorf("chmod %s failed: %w", file.archivePath, err))
		}
		if err := os.MkdirAll(filepath.Dir(file.targetPath), 0o755); err != nil {
			return restoreInstalledFiles(installed, fmt.Errorf("create target directory for %s failed: %w", file.archivePath, err))
		}

		backupPath := file.targetPath + ".backup"
		if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
			return restoreInstalledFiles(installed, fmt.Errorf("remove old backup for %s failed: %w", file.archivePath, err))
		}
		hadOriginal := true
		if err := os.Rename(file.targetPath, backupPath); err != nil {
			if !os.IsNotExist(err) {
				return restoreInstalledFiles(installed, fmt.Errorf("backup %s failed: %w", file.archivePath, err))
			}
			hadOriginal = false
		}

		if err := os.Rename(file.stagedPath, file.targetPath); err != nil {
			if hadOriginal {
				_ = os.Rename(backupPath, file.targetPath)
			}
			return restoreInstalledFiles(installed, fmt.Errorf("replace %s failed: %w", file.archivePath, err))
		}
		installed = append(installed, installedReleaseFile{
			targetPath:  file.targetPath,
			backupPath:  backupPath,
			hadOriginal: hadOriginal,
		})
	}
	return nil
}

func restoreInstalledFiles(installed []installedReleaseFile, cause error) error {
	for i := len(installed) - 1; i >= 0; i-- {
		file := installed[i]
		if err := os.Remove(file.targetPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("%w; restore remove failed for %s: %v", cause, file.targetPath, err)
		}
		if file.hadOriginal {
			if err := os.Rename(file.backupPath, file.targetPath); err != nil {
				return fmt.Errorf("%w; restore failed for %s: %v", cause, file.targetPath, err)
			}
		}
	}
	return cause
}

func restoreReleaseBackups(targets []string) error {
	files := make([]rollbackReleaseFile, 0, len(targets))
	for _, targetPath := range targets {
		backupPath := targetPath + ".backup"
		if _, err := os.Stat(backupPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("inspect rollback backup failed: %w", err)
		}
		files = append(files, rollbackReleaseFile{
			targetPath: targetPath,
			backupPath: backupPath,
			currentTmp: targetPath + ".rollback-current",
		})
	}

	restored := make([]rollbackReleaseFile, 0, len(files))
	for _, file := range files {
		if err := os.Remove(file.currentTmp); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("prepare rollback failed for %s: %w", file.targetPath, err)
		}
		if err := os.Rename(file.targetPath, file.currentTmp); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("stage current file for rollback failed: %w", err)
			}
		} else {
			file.hadCurrent = true
		}
		if err := os.Rename(file.backupPath, file.targetPath); err != nil {
			if file.hadCurrent {
				_ = os.Rename(file.currentTmp, file.targetPath)
			}
			return undoRestoredBackups(restored, fmt.Errorf("rollback failed for %s: %w", file.targetPath, err))
		}
		restored = append(restored, file)
	}

	for _, file := range restored {
		if !file.hadCurrent {
			continue
		}
		if err := os.Remove(file.currentTmp); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("rollback completed but cleanup failed for %s: %w", file.targetPath, err)
		}
	}
	return nil
}

func undoRestoredBackups(restored []rollbackReleaseFile, cause error) error {
	for i := len(restored) - 1; i >= 0; i-- {
		file := restored[i]
		if err := os.Rename(file.targetPath, file.backupPath); err != nil {
			return fmt.Errorf("%w; rollback recovery failed for %s: %v", cause, file.targetPath, err)
		}
		if file.hadCurrent {
			if err := os.Rename(file.currentTmp, file.targetPath); err != nil {
				return fmt.Errorf("%w; rollback recovery failed for %s: %v", cause, file.targetPath, err)
			}
		}
	}
	return cause
}

func (s *UpdateService) getFromCache(ctx context.Context) (*UpdateInfo, error) {
	data, err := s.cache.GetUpdateInfo(ctx)
	if err != nil {
		return nil, err
	}

	var cached struct {
		Latest                string       `json:"latest"`
		ReleaseInfo           *ReleaseInfo `json:"release_info"`
		UpstreamLatestVersion string       `json:"upstream_latest_version"`
		UpstreamWarning       string       `json:"upstream_warning"`
		Timestamp             int64        `json:"timestamp"`
	}
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, err
	}

	if time.Now().Unix()-cached.Timestamp > updateCacheTTL {
		return nil, fmt.Errorf("cache expired")
	}
	if cached.Latest == "" || cached.UpstreamLatestVersion == "" {
		return nil, fmt.Errorf("cache uses an obsolete release-channel format")
	}
	upstreamBaseline := strings.TrimPrefix(releasechannel.UpstreamBaseline, "v")

	return &UpdateInfo{
		CurrentVersion:        s.currentVersion,
		LatestVersion:         cached.Latest,
		HasUpdate:             compareVersions(s.currentVersion, cached.Latest) < 0,
		ReleaseInfo:           cached.ReleaseInfo,
		Cached:                true,
		BuildType:             s.buildType,
		ReleaseRepository:     githubRepo,
		ReleaseImage:          releasechannel.ReleaseImage,
		UpstreamRepository:    upstreamRepo,
		UpstreamBaseline:      upstreamBaseline,
		UpstreamLatestVersion: cached.UpstreamLatestVersion,
		UpstreamHasUpdate:     compareVersions(upstreamBaseline, cached.UpstreamLatestVersion) < 0,
		UpstreamWarning:       cached.UpstreamWarning,
	}, nil
}

func (s *UpdateService) saveToCache(ctx context.Context, info *UpdateInfo) {
	cacheData := struct {
		Latest                string       `json:"latest"`
		ReleaseInfo           *ReleaseInfo `json:"release_info"`
		UpstreamLatestVersion string       `json:"upstream_latest_version"`
		UpstreamWarning       string       `json:"upstream_warning"`
		Timestamp             int64        `json:"timestamp"`
	}{
		Latest:                info.LatestVersion,
		ReleaseInfo:           info.ReleaseInfo,
		UpstreamLatestVersion: info.UpstreamLatestVersion,
		UpstreamWarning:       info.UpstreamWarning,
		Timestamp:             time.Now().Unix(),
	}

	data, _ := json.Marshal(cacheData)
	_ = s.cache.SetUpdateInfo(ctx, string(data), time.Duration(updateCacheTTL)*time.Second)
}

// updateVersion is the release format used by this fork. The custom iteration
// is build metadata under SemVer, but it must affect update ordering here.
type updateVersion struct {
	base            [3]int
	customIteration int
}

// compareVersions compares base SemVer first, then the fork custom iteration.
func compareVersions(current, latest string) int {
	currentVersion, currentOK := parseVersion(current)
	latestVersion, latestOK := parseVersion(latest)

	if !currentOK || !latestOK {
		switch {
		case currentOK:
			return 1
		case latestOK:
			return -1
		default:
			return 0
		}
	}

	for i := 0; i < 3; i++ {
		if currentVersion.base[i] < latestVersion.base[i] {
			return -1
		}
		if currentVersion.base[i] > latestVersion.base[i] {
			return 1
		}
	}
	if currentVersion.customIteration < latestVersion.customIteration {
		return -1
	}
	if currentVersion.customIteration > latestVersion.customIteration {
		return 1
	}
	return 0
}

func parseVersion(v string) (updateVersion, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(v, "+")
	if len(parts) > 2 || parts[0] == "" {
		return updateVersion{}, false
	}

	baseParts := strings.Split(parts[0], ".")
	if len(baseParts) != 3 {
		return updateVersion{}, false
	}

	result := updateVersion{}
	for i, part := range baseParts {
		if part == "" {
			return updateVersion{}, false
		}
		parsed, err := strconv.Atoi(part)
		if err != nil || parsed < 0 {
			return updateVersion{}, false
		}
		result.base[i] = parsed
	}

	if len(parts) == 1 {
		return result, true
	}

	const customPrefix = "custom."
	if !strings.HasPrefix(parts[1], customPrefix) {
		return updateVersion{}, false
	}
	iterationText := strings.TrimPrefix(parts[1], customPrefix)
	if iterationText == "" {
		return updateVersion{}, false
	}
	iteration, err := strconv.Atoi(iterationText)
	if err != nil || iteration <= 0 {
		return updateVersion{}, false
	}
	result.customIteration = iteration
	return result, true
}

func parseForkReleaseVersion(v string) (updateVersion, bool) {
	parsed, ok := parseVersion(v)
	return parsed, ok && parsed.customIteration > 0
}

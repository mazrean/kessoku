package llmsetup

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// File permission constants.
const (
	// DirMode is the permission for created directories (rwxr-xr-x).
	DirMode os.FileMode = 0o755
	// FileMode is the permission for installed files (rw-r--r--).
	FileMode os.FileMode = 0o644
)

// InstallFile installs a file to the target directory using atomic rename.
// It creates the directory if needed, writes to a temp file, then renames.
// Note: Does not fsync the directory, so durability is not guaranteed on crash.
func InstallFile(targetDir string, fileName string, content []byte) (retErr error) {
	if err := os.MkdirAll(targetDir, DirMode); err != nil {
		return fmt.Errorf("cannot create directory: %s: %w", targetDir, err)
	}

	tmp, err := os.CreateTemp(targetDir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpName := tmp.Name()
	closed := false

	defer func() {
		if !closed {
			_ = tmp.Close()
		}
		if retErr != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		return fmt.Errorf("cannot write file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("cannot sync file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("cannot close temp file: %w", err)
	}
	closed = true

	if err := os.Chmod(tmpName, FileMode); err != nil {
		return fmt.Errorf("cannot set file permissions: %w", err)
	}

	finalPath := filepath.Join(targetDir, fileName)

	if err := os.Rename(tmpName, finalPath); err != nil {
		return fmt.Errorf("cannot install file %s: %w", finalPath, err)
	}

	return nil
}

// Install performs a complete Skills installation for the given agent.
// Skills directory (SKILL.md and support files/subdirectories) is installed recursively.
func Install(agent Agent, customPath string, userFlag bool) (string, error) {
	basePath, err := ResolvePath(customPath, userFlag, agent)
	if err != nil {
		return "", err
	}

	if err = ValidatePath(basePath); err != nil {
		return "", err
	}

	skillPath := filepath.Join(basePath, agent.SkillsDirName())
	skillsFS := agent.SkillsFS()
	srcDir := agent.SkillsSrcDir()

	// Verify srcDir exists before walking
	if _, err = fs.Stat(skillsFS, srcDir); err != nil {
		return "", fmt.Errorf("cannot read skills directory: %w", err)
	}

	// Walk the embedded directory tree and install all files
	err = fs.WalkDir(skillsFS, srcDir, func(fsPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip directories (they're created automatically by InstallFile)
		if d.IsDir() {
			return nil
		}

		// Calculate relative path from srcDir
		relPath, relErr := filepath.Rel(srcDir, fsPath)
		if relErr != nil {
			return fmt.Errorf("cannot compute relative path for %s: %w", fsPath, relErr)
		}

		content, readErr := fs.ReadFile(skillsFS, fsPath)
		if readErr != nil {
			return fmt.Errorf("cannot read embedded file %s: %w", fsPath, readErr)
		}

		// Determine target directory (skillPath + parent directories of relPath)
		targetDir := filepath.Join(skillPath, filepath.Dir(relPath))
		fileName := filepath.Base(relPath)

		if installErr := InstallFile(targetDir, fileName, content); installErr != nil {
			return installErr
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return skillPath, nil
}

package db

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// EnsureExtensionFiles checks and automatically deploys native pgvector extension binaries into embedded postgres runtime (§7.2, §12).
func EnsureExtensionFiles(dbPath string) error {
	extDir := filepath.Join(dbPath, "binaries", "share", "extension")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		return fmt.Errorf("failed creating extension directory %s: %w", extDir, err)
	}

	libDir := filepath.Join(dbPath, "binaries", "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return fmt.Errorf("failed creating lib directory %s: %w", libDir, err)
	}

	// Source directory for extracted vendor binaries
	vendorDir := filepath.Join("pkg", "db", "vendor_extensions", "windows_amd64", "extracted")

	// 1. Copy vector.dll if present in vendor_extensions
	vendorDLL := filepath.Join(vendorDir, "lib", "vector.dll")
	targetDLL := filepath.Join(libDir, "vector.dll")

	if _, err := os.Stat(vendorDLL); err == nil {
		if err := copyFile(vendorDLL, targetDLL); err != nil {
			log.Printf("[Zuri Extension] Warning: Failed deploying vector.dll: %v", err)
		} else {
			log.Printf("[Zuri Extension] Deployed native vector.dll to %s", targetDLL)
		}
	}

	// 2. Copy extension manifest files (.control and .sql)
	vendorExtDir := filepath.Join(vendorDir, "share", "extension")
	deployedCount := 0
	if entries, err := os.ReadDir(vendorExtDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			srcFile := filepath.Join(vendorExtDir, entry.Name())
			dstFile := filepath.Join(extDir, entry.Name())
			if err := copyFile(srcFile, dstFile); err == nil {
				deployedCount++
			}
		}
		log.Printf("[Zuri Extension] Deployed native pgvector package (vector.dll + %d extension scripts)", deployedCount)
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

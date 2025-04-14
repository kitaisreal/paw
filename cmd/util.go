package main

import (
	"io"
	"os"

	"github.com/kitaisreal/paw/internal/logger"
)

func createDirectoryOrExit(path string) {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		logger.Log.Errorf("Failed to create directory %s: %v", path, err)
		os.Exit(1)
	}
}

func removeDirectoryOrExit(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		logger.Log.Errorf("Failed to remove directory %s: %v", path, err)
		os.Exit(1)
	}
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

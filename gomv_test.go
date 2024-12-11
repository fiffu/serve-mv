package main

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func withTemp(t *testing.T, dir, fileName string, callback func(tmpDir string, tmpFile *os.File)) {
	tmpRoot := os.TempDir()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	tmpDirPath := path.Join(tmpRoot, dir)
	if err := os.MkdirAll(tmpDirPath, 0777); err != nil {
		t.Fatalf("Failed to create %s with err: %v", tmpDirPath, err)
	}

	file, err := os.CreateTemp(tmpDirPath, fileName)
	if err != nil {
		t.Fatalf("Failed to create %s/%s with err: %v", tmpDirPath, fileName, err)
	}
	tmpFilePath := file.Name()
	defer os.Remove(tmpFilePath)
	defer os.Remove(tmpDirPath)

	callback(tmpDirPath, file)
}

func Test_NewSystemJSON(t *testing.T) {
	withTemp(t, path.Join("www", "data"), "System.json",
		func(tmpDir string, tmpFile *os.File) {
			_, err := tmpFile.Write([]byte(
				`{"gameTitle", "Hello World"}`,
			))
			assert.NoError(t, err)

			sj, err := NewSystemJSON(tmpDir)
			assert.NoError(t, err)

			assert.Equal(t, "Hello World", sj.GameTitle)
		},
	)
}

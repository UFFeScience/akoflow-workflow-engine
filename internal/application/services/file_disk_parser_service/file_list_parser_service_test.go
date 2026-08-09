package file_list_parser_service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFileListPreservesDirectoryAndMetadata(t *testing.T) {
	raw := "./input:\n-rw-r--r-- 1 user group 42 Jan 02 03:04 file.txt\ninvalid"
	service := New()
	encoded := service.Parse(raw)
	var files []FileDisk
	require.NoError(t, json.Unmarshal([]byte(encoded), &files))
	require.Equal(t, []FileDisk{{
		Permissions: "-rw-r--r--", Owner: "user", Group: "group", Size: "42",
		LastModified: "Jan 02 03:04", Name: "file.txt", Path: "./input",
	}}, files)
}

func TestParseEmptyFileListReturnsEmptyJSONList(t *testing.T) {
	service := New()
	require.JSONEq(t, `[]`, service.Parse(""))
}

package file_spec_parser_service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDiskSpecSkipsHeaderAndInvalidRows(t *testing.T) {
	raw := "Filesystem Size Used Avail Use% Mounted on\n/dev/sda1 100G 20G 80G 20% /data\nbad"
	service := New()
	encoded := service.Parse(raw)
	var specs []DiskSpec
	require.NoError(t, json.Unmarshal([]byte(encoded), &specs))
	require.Equal(t, []DiskSpec{{
		FileSystem: "/dev/sda1", Size: "100G", Used: "20G", Available: "80G",
		UsedPercentage: "20%", MountedOn: "/data",
	}}, specs)
}

func TestParseDiskSpecWithOnlyHeaderReturnsEmptyList(t *testing.T) {
	service := New()
	require.JSONEq(t, `[]`, service.Parse("header"))
}

package slurm

import "testing"

func TestParseDiscoveryBuildsUniqueNodesWithPartitionMembership(t *testing.T) {
	result := parseDiscovery([]byte(`FACT|architecture|x86_64
PARTITION|routage*|up|2|48|192000|3-00:00:00
PARTITION|preempt|up|1|48|192000|3-00:00:00
NODE|routage*|bora001|idle|48|192000|1000|1|avx2|gpu:a100:2|none
NODE|preempt|bora001|idle|48|192000|1000|1|avx2|gpu:a100:2|none
NODE|routage*|bora002|alloc|48|192000|1000|1|(null)|(null)|job running
FILESYSTEM|/dev/home|lustre|1000|200|800|/home
`))
	partitions := result.Metadata["partitions"].([]map[string]any)
	nodes := result.Metadata["nodes"].([]map[string]any)
	if !result.Available || len(partitions) != 2 || len(nodes) != 2 {
		t.Fatalf("result=%+v", result)
	}
	if partitions[0]["name"] != "routage" || partitions[0]["default"] != true {
		t.Fatalf("partition=%+v", partitions[0])
	}
	membership := nodes[0]["partitions"].([]string)
	if len(membership) != 2 || membership[0] != "routage" || membership[1] != "preempt" {
		t.Fatalf("node=%+v", nodes[0])
	}
	if nodes[0]["cpuCores"] != int64(48) || nodes[0]["memoryMiB"] != int64(192000) {
		t.Fatalf("node capacities=%+v", nodes[0])
	}
}

func TestParseDiscoveryWarnsAboutMalformedRecords(t *testing.T) {
	result := parseDiscovery([]byte("PARTITION|broken\nNODE|broken\n"))
	if result.Available || len(result.Warnings) != 2 {
		t.Fatalf("result=%+v", result)
	}
}

package schema

import _ "embed"

const Version = 5

// SQL is the single canonical database definition used by production and tests.
//
//go:embed schema.sql
var SQL string

package models

import (
	"fmt"
	"os"
	"sync"

	"zid/idcodec"
)

// ZID codec initialization
var (
	globalCodec *idcodec.IDCodec
	once        sync.Once
	initErr     error
)

// EntityMeta defines metadata for each entity/table
type EntityMeta struct {
	Table       string
	Prefix      string
	Model       any
	AutoMigrate bool
}

// Entities is the registry of all entities in the system
var Entities = []EntityMeta{
	{Table: "providers", Prefix: "PROV", Model: &Provider{}, AutoMigrate: true},
	{Table: "skills", Prefix: "SKIL", Model: &Skill{}, AutoMigrate: true},
	{Table: "scan_jobs", Prefix: "SCAN", Model: &ScanJob{}, AutoMigrate: true},
	{Table: "scan_issues", Prefix: "SISS", Model: &ScanIssue{}, AutoMigrate: true},
}

// initZID initializes the global ID codec
func initZID() error {
	once.Do(func() {
		masterKey := os.Getenv("ZID_MASTER_KEY")
		if masterKey == "" {
			masterKey = "default-dev-key-change-in-production-32bytes"
		}
		tweak := os.Getenv("ZID_TWEAK")
		if tweak == "" {
			tweak = "v1|app|dev"
		}
		codec, err := idcodec.New([]byte(masterKey), []byte(tweak))
		if err != nil {
			initErr = fmt.Errorf("failed to initialize ID codec: %w", err)
			return
		}
		globalCodec = codec
	})
	return initErr
}

// Encode generates a zid from prefix and numeric ID
func Encode(prefix string, id uint64) (string, error) {
	if globalCodec == nil {
		if err := initZID(); err != nil {
			return "", err
		}
	}
	return globalCodec.Encrypt(prefix, id)
}

// MustEncode panics on error
func MustEncode(prefix string, id uint64) string {
	z, err := Encode(prefix, id)
	if err != nil {
		panic(err)
	}
	return z
}

// Decode parses a zid and returns prefix and numeric ID
func Decode(zid string) (string, uint64, error) {
	if globalCodec == nil {
		if err := initZID(); err != nil {
			return "", 0, err
		}
	}
	return globalCodec.Decrypt(zid)
}

// MustDecode panics on error
func MustDecode(zid string) (string, uint64) {
	p, id, err := Decode(zid)
	if err != nil {
		panic(err)
	}
	return p, id
}

// TablePrefix maps table names to ZID prefixes
var TablePrefix map[string]string

// GetPrefixForTable returns the ZID prefix for a table name
func GetPrefixForTable(tableName string) string {
	return TablePrefix[tableName]
}

// ModelsForAutoMigrate is the list of models for DB migrations
var ModelsForAutoMigrate []interface{}

// CleanupTablesInOrder lists tables to truncate in FK-safe order
var CleanupTablesInOrder []string

// init builds derived registries from Entities
func init() {
	TablePrefix = make(map[string]string, len(Entities))
	for _, e := range Entities {
		if e.Prefix != "" {
			TablePrefix[e.Table] = e.Prefix
		}
	}

	for _, e := range Entities {
		CleanupTablesInOrder = append(CleanupTablesInOrder, e.Table)
	}

	for _, e := range Entities {
		if e.AutoMigrate && e.Model != nil {
			ModelsForAutoMigrate = append(ModelsForAutoMigrate, e.Model)
		}
	}
}

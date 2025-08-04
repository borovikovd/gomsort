package testdata

import "fmt"

// Database represents a database connection
type Database struct {
	host string
	port int
}

// Complex example with various method types and call patterns

// Helper method called by multiple methods (high in-degree)

// Deep helper method (high depth)

// Entry point method (low depth, exported)

// Private helper with medium depth

// Another entry point

// Deepest level helper

// Medium level helper

// Another deep helper

// Entry point method (exported)

// Helper for Close

// Row represents a database row
type Row struct {
	data map[string]interface{}
}

// Simple method with no dependencies

// Method that calls another method

// Helper method

// Another entry point

// Cache represents an in-memory cache
type Cache struct {
	items map[string]interface{}
}

// Entry point
func (c *Cache) Get(key string) (interface{}, bool) {
	if !c.isValid() {
		return nil, false
	}
	return c.retrieve(key)
}

// Helper with medium depth

// Shared helper (high in-degree)

// Entry point
func (c *Cache) Set(key string, value interface{}) error {
	if !c.isValid() {
		c.initialize()
	}
	c.store(key, value)
	return nil
}

func (c *Cache) isValid() bool {
	return c.items != nil
}

func (c *Cache) retrieve(key string) (interface{}, bool) {
	val, ok := c.items[key]
	return val, ok
}

// Helper for Set
func (c *Cache) store(key string, value interface{}) {
	c.items[key] = value
}

// Helper for initialization
func (c *Cache) initialize() {
	c.items = make(map[string]interface{})
}

func (db *Database) Connect() error {
	if err := db.validateConnection(); err != nil {
		return err
	}
	return db.establishConnection()
}

func (db *Database) Query(sql string) ([]Row, error) {
	if err := db.validateConnection(); err != nil {
		return nil, err
	}
	data, err := db.executeRawQuery(sql)
	if err != nil {
		return nil, err
	}
	return db.parseResults(data)
}

func (db *Database) Close() error {
	return db.cleanup()
}

func (db *Database) validateConnection() error {
	if db.host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	return nil
}

func (db *Database) executeRawQuery(query string) ([]byte, error) {
	return db.lowLevelExecute(query)
}

func (db *Database) establishConnection() error {
	return db.performHandshake()
}

func (db *Database) lowLevelExecute(query string) ([]byte, error) {
	return []byte("result"), nil
}

func (db *Database) parseResults(data []byte) ([]Row, error) {
	return []Row{}, nil
}

func (db *Database) performHandshake() error {
	return nil
}

func (db *Database) cleanup() error {
	return nil
}

func (r *Row) GetString(key string) string {
	if val, ok := r.data[key].(string); ok {
		return val
	}
	return ""
}

func (r *Row) HasKey(key string) bool {
	_, exists := r.data[key]
	return exists
}

func (r *Row) GetValue(key string) interface{} {
	return r.getValue(key)
}

func (r *Row) getValue(key string) interface{} {
	return r.data[key]
}

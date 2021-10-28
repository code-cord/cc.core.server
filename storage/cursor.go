package storage

import (
	"github.com/boltdb/bolt"
)

// Cursor represents storage cursor for navigating across data.
type Cursor struct {
	tx     *bolt.Tx
	cursor *bolt.Cursor
}

// Next moves cursor to the next item in the bucket.
func (c *Cursor) Next() (RawValue, bool) {
	key, value := c.cursor.Next()

	return &rawValue{
		v: value,
	}, key != nil
}

// First moves the cursor to the first item in the bucket.
func (c *Cursor) First() (RawValue, bool) {
	key, value := c.cursor.First()

	return &rawValue{
		v: value,
	}, key != nil
}

// Close closes cursor.
func (c *Cursor) Close() error {
	return c.tx.Rollback()
}

package storage

import (
	"fmt"

	"github.com/boltdb/bolt"
)

// Bucket represents storage bucket model.
type Bucket struct {
	name string
	db   *bolt.DB
}

// RawValue represents interface for working with the storage raw value.
type RawValue interface {
	Decode(out interface{}, unmarshalFn UnmarshalFn) error
	Get() []byte
}

// MarshalFn represents type for the marshal value func.
type MarshalFn = func(v interface{}) ([]byte, error)

// UnmarshalFn represents type for the unmarshal value func.
type UnmarshalFn = func(v []byte, out interface{}) error

type rawValue struct {
	v []byte
}

// Size returns number of key-value pairs in the bucket.
func (b *Bucket) Size() (s int) {
	b.db.View(func(t *bolt.Tx) error {
		s = t.Bucket(wrap(b.name)).Stats().KeyN

		return nil
	})

	return
}

// Load returns value by key.
//
// If key doesn't exist it returns nil.
func (b *Bucket) Load(key string) (rv RawValue) {
	b.db.View(func(t *bolt.Tx) error {
		bucket := t.Bucket(wrap(b.name))

		v := bucket.Get(wrap(key))
		if v != nil {
			rv = &rawValue{
				v: v,
			}
		}

		return nil
	})

	return
}

// All fetches all bucket's data.
func (b *Bucket) All() (*Cursor, error) {
	tx, err := b.db.Begin(false)
	if err != nil {
		return nil, fmt.Errorf("could not start reading transaction: %v", err)
	}

	bucket := tx.Bucket(wrap(b.name))
	return &Cursor{
		tx:     tx,
		cursor: bucket.Cursor(),
	}, nil
}

// Store stores a new key-value pair in the storage bucket.
//
// Before storing, it encodes the value using the provided marshal function.
// If no marshaling function is specified, the default marshaler will be used.
func (b *Bucket) Store(key string, value interface{}, marshalFn MarshalFn) error {
	return b.db.Update(func(t *bolt.Tx) error {
		if marshalFn == nil {
			marshalFn = defaultMarshalFn
		}

		v, err := marshalFn(value)
		if err != nil {
			return fmt.Errorf("could not encode value: %v", err)
		}

		bucket := t.Bucket(wrap(b.name))
		return bucket.Put(wrap(key), v)
	})
}

// Delete deletes value by the key.
func (b *Bucket) Delete(key string) error {
	return b.db.Update(func(t *bolt.Tx) error {
		bucket := t.Bucket(wrap(b.name))
		return bucket.Delete(wrap(key))
	})
}

// Decode decodes raw value into out argument using provided unmarshal func.
func (r *rawValue) Decode(out interface{}, unmarshalFn UnmarshalFn) error {
	return unmarshalFn(r.v, out)
}

// Get returns raw value's data.
func (r *rawValue) Get() []byte {
	return r.v
}

func defaultMarshalFn(v interface{}) ([]byte, error) {
	value, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid value type: expected []byte, actual: %T", v)
	}

	return value, nil
}

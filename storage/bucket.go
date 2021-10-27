package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/boltdb/bolt"
)

// Bucket represents storage bucket model.
type Bucket struct {
	name string
	db   *bolt.DB
}

// All fetches all bucket's data.
func (b *Bucket) All(out interface{}) error {
	// TODO: check not pointer
	if reflect.TypeOf(out).Elem().Kind() != reflect.Slice {
		return errors.New("`out` must be a slice")
	}

	return b.db.View(func(t *bolt.Tx) error {
		bucket := t.Bucket(wrap(b.name))

		return bucket.ForEach(func(k, v []byte) error {
			val := reflect.New(reflect.TypeOf(reflect.TypeOf(out).Elem())).Interface()
			err := json.Unmarshal(v, &val)
			if err != nil {
				return fmt.Errorf("could not unmarshal value: %v", err)
			}

			valuePtr := reflect.ValueOf(out)
			value := valuePtr.Elem()

			value.Set(reflect.Append(value, reflect.New(reflect.TypeOf(reflect.TypeOf(out).Elem()))))

			return nil
		})
	})
}

func (b *Bucket) Store(key string, value interface{}) error {
	return b.db.Update(func(t *bolt.Tx) error {
		v, err := json.Marshal(value)
		if err != nil {
			return err
		}

		bucket := t.Bucket(wrap(b.name))
		return bucket.Put(wrap(key), v)
	})
}

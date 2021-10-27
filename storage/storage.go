package storage

import (
	"fmt"
	"sync"

	"github.com/boltdb/bolt"
)

// Storage represents storage implementation model.
type Storage struct {
	db            *bolt.DB
	buckets       *sync.Map
	defaultBucket string
}

// Config represents storage configuration model.
type Config struct {
	DBPath        string
	Buckets       []string
	DefaultBucket string
}

type wrap []byte

// New returns new storage instance.
func New(cfg Config) (*Storage, error) {
	db, err := bolt.Open(cfg.DBPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %v", err)
	}

	buckets := new(sync.Map)
	err = db.Update(func(tx *bolt.Tx) error {
		for i := range cfg.Buckets {
			bucketName := cfg.Buckets[i]

			_, err := tx.CreateBucket(wrap(bucketName))
			if err != nil && err != bolt.ErrBucketExists {
				return fmt.Errorf("could not create `%s` bucket: %v", bucketName, err)
			}

			buckets.Store(bucketName, &Bucket{
				name: bucketName,
				db:   db,
			})
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not init storage buckets: %v", err)
	}

	return &Storage{
		db:            db,
		buckets:       buckets,
		defaultBucket: cfg.DefaultBucket,
	}, nil
}

// Close closes storage.
func (s *Storage) Close() error {
	return s.db.Close()
}

// Use returns pointer to the storage bucket by name.
//
// If bucket doesn't exists it returns nil.
func (s *Storage) Use(bucketName string) *Bucket {
	bucket, ok := s.buckets.Load(bucketName)
	if !ok {
		return nil
	}

	return bucket.(*Bucket)
}

// Default returns pointer to the default storage bucket.
//
// If bucket doesn't exists it returns nil.
func (s *Storage) Default() *Bucket {
	bucket, ok := s.buckets.Load(s.defaultBucket)
	if !ok {
		return nil
	}

	return bucket.(*Bucket)
}

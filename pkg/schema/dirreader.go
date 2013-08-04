/*
Copyright 2011 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"camlistore.org/pkg/blob"
)

// A DirReader reads the entries of a "directory" schema blob's
// referenced "static-set" blob.
type DirReader struct {
	fetcher blob.SeekFetcher
	ss      *superset

	staticSet []blob.Ref
	current   int
}

// NewDirReader creates a new directory reader and prepares to
// fetch the static-set entries
func NewDirReader(fetcher blob.SeekFetcher, dirBlobRef blob.Ref) (*DirReader, error) {
	ss := new(superset)
	err := ss.setFromBlobRef(fetcher, dirBlobRef)
	if err != nil {
		return nil, err
	}
	if ss.Type != "directory" {
		return nil, fmt.Errorf("schema/filereader: expected \"directory\" schema blob for %s, got %q", dirBlobRef, ss.Type)
	}
	dr, err := ss.NewDirReader(fetcher)
	if err != nil {
		return nil, fmt.Errorf("schema/filereader: creating DirReader for %s: %v", dirBlobRef, err)
	}
	dr.current = 0
	return dr, nil
}

func (b *Blob) NewDirReader(fetcher blob.SeekFetcher) (*DirReader, error) {
	return b.ss.NewDirReader(fetcher)
}

func (ss *superset) NewDirReader(fetcher blob.SeekFetcher) (*DirReader, error) {
	if ss.Type != "directory" {
		return nil, fmt.Errorf("Superset not of type \"directory\"")
	}
	return &DirReader{fetcher: fetcher, ss: ss}, nil
}

func (ss *superset) setFromBlobRef(fetcher blob.SeekFetcher, blobRef blob.Ref) error {
	if !blobRef.Valid() {
		return errors.New("schema/filereader: blobref invalid")
	}
	ss.BlobRef = blobRef
	rsc, _, err := fetcher.Fetch(blobRef)
	if err != nil {
		return fmt.Errorf("schema/filereader: fetching schema blob %s: %v", blobRef, err)
	}
	defer rsc.Close()
	if err = json.NewDecoder(rsc).Decode(ss); err != nil {
		return fmt.Errorf("schema/filereader: decoding schema blob %s: %v", blobRef, err)
	}
	return nil
}

// StaticSet returns the whole of the static set members of that directory
func (dr *DirReader) StaticSet() ([]blob.Ref, error) {
	if dr.staticSet != nil {
		return dr.staticSet, nil
	}
	staticSetBlobref := dr.ss.Entries
	if !staticSetBlobref.Valid() {
		return nil, fmt.Errorf("schema/filereader: Invalid blobref\n")
	}
	rsc, _, err := dr.fetcher.Fetch(staticSetBlobref)
	if err != nil {
		return nil, fmt.Errorf("schema/filereader: fetching schema blob %s: %v", staticSetBlobref, err)
	}
	ss, err := parseSuperset(rsc)
	if err != nil {
		return nil, fmt.Errorf("schema/filereader: decoding schema blob %s: %v", staticSetBlobref, err)
	}
	if ss.Type != "static-set" {
		return nil, fmt.Errorf("schema/filereader: expected \"static-set\" schema blob for %s, got %q", staticSetBlobref, ss.Type)
	}
	for _, member := range ss.Members {
		if !member.Valid() {
			return nil, fmt.Errorf("schema/filereader: invalid (static-set member) blobref\n")
		}
		dr.staticSet = append(dr.staticSet, member)
	}
	return dr.staticSet, nil
}

// Readdir implements the Directory interface.
func (dr *DirReader) Readdir(n int) (entries []DirectoryEntry, err error) {
	sts, err := dr.StaticSet()
	if err != nil {
		return nil, fmt.Errorf("schema/filereader: can't get StaticSet: %v\n", err)
	}
	up := dr.current + n
	if n <= 0 {
		dr.current = 0
		up = len(sts)
	} else {
		if n > (len(sts) - dr.current) {
			err = io.EOF
			up = len(sts)
		}
	}

	// TODO(bradfitz): push down information to the fetcher
	// (e.g. cachingfetcher -> remote client http) that we're
	// going to load a bunch, so the HTTP client (if not using
	// SPDY) can do discovery and see if the server supports a
	// batch handler, then get them all in one round-trip, rather
	// than attacking the server with hundreds of parallel TLS
	// setups.

	type res struct {
		ent DirectoryEntry
		err error
	}
	var cs []chan res

	// Kick off all directory entry loads.
	// TODO: bound this?
	for _, entRef := range sts[dr.current:up] {
		c := make(chan res, 1)
		cs = append(cs, c)
		go func(entRef blob.Ref) {
			entry, err := NewDirectoryEntryFromBlobRef(dr.fetcher, entRef)
			c <- res{entry, err}
		}(entRef)
	}

	for _, c := range cs {
		res := <-c
		if res.err != nil {
			return nil, fmt.Errorf("schema/filereader: can't create dirEntry: %v\n", err)
		}
		entries = append(entries, res.ent)
	}
	return entries, nil
}

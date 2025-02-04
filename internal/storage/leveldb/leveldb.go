package leveldb

import (
	"context"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"sort"
	"strconv"
	"strings"
)

type Storage struct {
	db *leveldb.DB
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func NewStorage(path string) (*Storage, error) {
	const op = "storage.leveldb.New"

	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) AddDocument(context context.Context, content string) (int, error) {
	batch := new(leveldb.Batch)

	// Retrieve the last document ID
	lastIDBytes, err := s.db.Get([]byte("doc_counter"), nil)
	var lastID int
	if err == nil {
		lastID, _ = strconv.Atoi(string(lastIDBytes))
	}

	newID := lastID + 1
	newIDStr := strconv.Itoa(newID)

	// Update document counter
	batch.Put([]byte("doc_counter"), []byte(newIDStr))

	// Save the document content
	batch.Put([]byte("doc:"+newIDStr), []byte(content))

	// Word indexing
	wordsCount := make(map[string]int)
	words := tokenizeWords(content)
	for _, word := range words {
		wordsCount[word]++
	}

	for word, count := range wordsCount {
		wordKey := "word:" + word
		existing, err := s.db.Get([]byte(wordKey), nil)

		var indexData string
		if err == nil && len(existing) > 0 {
			indexData = string(existing) + ","
		}

		indexData += fmt.Sprintf("%d:%d", newID, count) // append the new index

		fmt.Printf("Saving new data for word %s, %d:%d\n", wordKey, newID, count)

		// Save the updated index data for the word
		batch.Put([]byte(wordKey), []byte(indexData))
	}

	// Apply all batch operations
	fmt.Printf("Saving new document batch (document_count:%s; document_id:%s; document_value:%s\n", newIDStr, "doc:"+newIDStr, content)

	err = s.db.Write(batch, nil)
	if err != nil {
		return 0, err
	}

	return newID, nil
}

func (s *Storage) Search(context context.Context, word string) ([]string, error) {
	wordKey := "word:" + word
	data, err := s.db.Get([]byte(wordKey), nil)
	if err != nil {
		return nil, fmt.Errorf("word %s not found", word)
	}

	// Parse the stored index data (word = docID:count pairs)
	var results []struct {
		docID int
		count int
	}

	// Split entries by comma and parse each "docID:count" pair
	pairs := strings.Split(string(data), ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			continue // Skip invalid entries
		}
		docID, _ := strconv.Atoi(parts[0])
		count, _ := strconv.Atoi(parts[1])
		results = append(results, struct {
			docID int
			count int
		}{docID, count})
	}

	// Sort results by count (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].count > results[j].count
	})

	// Fetch document data for each result
	var docs []string
	for _, r := range results {
		docKey := "doc:" + strconv.Itoa(r.docID)
		docData, err := s.db.Get([]byte(docKey), nil)
		if err == nil {
			docs = append(docs, fmt.Sprintf("Doc %d (x%d): %s", r.docID, r.count, docData))
		}
	}

	return docs, nil
}

func (s *Storage) DeleteDocument(context context.Context, docId int) error {
	batch := new(leveldb.Batch)

	docKey := "doc:" + strconv.Itoa(docId)
	batch.Delete([]byte(docKey))

	// Run over all indexes and delete references to document
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		key := string(iter.Key())

		if strings.HasPrefix(key, "word:") {
			value := string(iter.Value())
			entries := strings.Split(value, ",")
			var newEntries []string
			for _, entry := range entries {
				parts := strings.Split(entry, ":")
				id, _ := strconv.Atoi(parts[0])
				if id != docId {
					newEntries = append(newEntries, entry)
				}
			}

			// If word is in other documents - update, otherwise delete
			if len(newEntries) > 0 {
				batch.Put([]byte(key), []byte(strings.Join(newEntries, ",")))
			} else {
				batch.Delete([]byte(key))
			}
		}
	}

	iter.Release()

	return s.db.Write(batch, nil)
}

func tokenizeWords(content string) []string {
	//TODO Add logic for tokenizing words
	return strings.Fields(content)
}

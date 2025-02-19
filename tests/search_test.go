package tests

import (
	"context"
	"fts-hw/internal/services/fts"
	"fts-hw/internal/storage/leveldb"
	"log/slog"
	"os"
	"testing"
)

func TestSearch(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	storage, err := leveldb.NewStorage("./storage/leveldb.db")
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	searchEngine := fts.New(log, storage, storage)

	content := "Search engine test document."
	_, err = searchEngine.AddDocument(context.Background(), content)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	results, err := searchEngine.Search(context.Background(), "test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("Expected search results, but got 0")
	}

	t.Logf("Search returned %d results", len(results))
	for _, result := range results {
		t.Log(result)
	}
}

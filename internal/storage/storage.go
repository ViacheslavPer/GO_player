package storage

// Storage is in-memory MVP storage.
// No persistence. No Badger. No files.
type Storage interface {
	// TODO: Song CRUD (MVP minimal)
	// TODO: BaseGraph load/save (in-memory)
}

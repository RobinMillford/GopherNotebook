package notebook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager provides CRUD operations for notebooks, stored as JSON on disk.
type Manager struct {
	dataDir string
	mu      sync.RWMutex
}

// NewManager creates a notebook manager writing to the given directory.
func NewManager(dataDir string) (*Manager, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create notebook data dir: %w", err)
	}
	return &Manager{dataDir: dataDir}, nil
}

// List returns all notebooks.
func (m *Manager) List() ([]Notebook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, err := os.ReadDir(m.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read notebook dir: %w", err)
	}

	var notebooks []Notebook
	for _, entry := range entries {
		if entry.IsDir() {
			nb, err := m.load(entry.Name())
			if err != nil {
				continue // Skip corrupted entries
			}
			notebooks = append(notebooks, *nb)
		}
	}

	return notebooks, nil
}

// Get returns a specific notebook by ID.
func (m *Manager) Get(id string) (*NotebookDetail, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nb, err := m.load(id)
	if err != nil {
		return nil, err
	}

	sources, err := m.loadSources(id)
	if err != nil {
		sources = []Source{} // Graceful fallback
	}

	messages, err := m.loadMessages(id)
	if err != nil {
		messages = []Message{} // Graceful fallback
	}

	return &NotebookDetail{
		Notebook: *nb,
		Sources:  sources,
		Messages: messages,
	}, nil
}

// Create creates a new notebook and returns it.
func (m *Manager) Create(name, description string) (*Notebook, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	nb := &Notebook{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		FileCount:   0,
	}

	if err := m.save(nb); err != nil {
		return nil, err
	}

	return nb, nil
}

// Delete removes a notebook and its metadata.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dir := filepath.Join(m.dataDir, id)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("notebook %s not found", id)
	}

	return os.RemoveAll(dir)
}

// AddSource records a source file in the notebook's metadata.
func (m *Manager) AddSource(notebookID string, source Source) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sources, err := m.loadSources(notebookID)
	if err != nil {
		sources = []Source{}
	}

	// Update or append
	found := false
	for i, s := range sources {
		if s.FileName == source.FileName {
			sources[i] = source
			found = true
			break
		}
	}
	if !found {
		sources = append(sources, source)
	}

	// Update notebook file count
	nb, err := m.load(notebookID)
	if err != nil {
		return err
	}
	nb.FileCount = len(sources)
	nb.UpdatedAt = time.Now()

	if err := m.save(nb); err != nil {
		return err
	}

	return m.saveSources(notebookID, sources)
}

// RemoveSource removes a source from the notebook's metadata.
func (m *Manager) RemoveSource(notebookID, fileName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure fileName is safe (no path traversal)
	fileName = filepath.Base(fileName)

	sources, err := m.loadSources(notebookID)
	if err != nil {
		return err
	}

	var filtered []Source
	for _, s := range sources {
		if s.FileName != fileName {
			filtered = append(filtered, s)
		}
	}

	nb, err := m.load(notebookID)
	if err != nil {
		return err
	}
	nb.FileCount = len(filtered)
	nb.UpdatedAt = time.Now()

	if err := m.save(nb); err != nil {
		return err
	}

	return m.saveSources(notebookID, filtered)
}

// GetSources returns the list of sources for a notebook.
func (m *Manager) GetSources(notebookID string) ([]Source, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loadSources(notebookID)
}

// --- Internal file operations ---

func (m *Manager) notebookDir(id string) string {
	return filepath.Join(m.dataDir, id)
}

func (m *Manager) notebookFile(id string) string {
	return filepath.Join(m.notebookDir(id), "notebook.json")
}

func (m *Manager) sourcesFile(id string) string {
	return filepath.Join(m.notebookDir(id), "sources.json")
}

func (m *Manager) messagesFile(id string) string {
	return filepath.Join(m.notebookDir(id), "messages.json")
}

// GetMessages returns the chat history for a notebook.
func (m *Manager) GetMessages(notebookID string) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loadMessages(notebookID)
}

// loadMessages reads the messages file from disk. Caller must hold at least a read lock.
func (m *Manager) loadMessages(notebookID string) ([]Message, error) {
	data, err := os.ReadFile(m.messagesFile(notebookID))
	if err != nil {
		if os.IsNotExist(err) {
			return []Message{}, nil
		}
		return nil, err
	}

	var messages []Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// SaveMessage appends a message to the notebook's chat history.
func (m *Manager) SaveMessage(notebookID string, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	messages, err := m.loadMessages(notebookID)
	if err != nil {
		messages = []Message{}
	}

	messages = append(messages, msg)

	// Update notebook last updated time
	if nb, err := m.load(notebookID); err == nil {
		nb.UpdatedAt = time.Now()
		_ = m.save(nb)
	}

	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.messagesFile(notebookID), data, 0644)
}

func (m *Manager) load(id string) (*Notebook, error) {
	data, err := os.ReadFile(m.notebookFile(id))
	if err != nil {
		return nil, fmt.Errorf("notebook %s not found: %w", id, err)
	}

	var nb Notebook
	if err := json.Unmarshal(data, &nb); err != nil {
		return nil, fmt.Errorf("corrupted notebook %s: %w", id, err)
	}
	return &nb, nil
}

func (m *Manager) save(nb *Notebook) error {
	dir := m.notebookDir(nb.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(nb, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.notebookFile(nb.ID), data, 0644)
}

func (m *Manager) loadSources(id string) ([]Source, error) {
	data, err := os.ReadFile(m.sourcesFile(id))
	if err != nil {
		if os.IsNotExist(err) {
			return []Source{}, nil
		}
		return nil, err
	}

	var sources []Source
	if err := json.Unmarshal(data, &sources); err != nil {
		return nil, err
	}
	return sources, nil
}

func (m *Manager) saveSources(id string, sources []Source) error {
	data, err := json.MarshalIndent(sources, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.sourcesFile(id), data, 0644)
}

// ClearChatHistory removes the messages file for a notebook.
func (m *Manager) ClearChatHistory(notebookID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	file := m.messagesFile(notebookID)
	if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear chat history for %s: %w", notebookID, err)
	}
	return nil
}

package partition

import (
	"fmt"
	"sync"
	"time"
)

// HistoryEntry represents a single operation in the history
type HistoryEntry struct {
	ID          int
	Timestamp   time.Time
	Operation   string
	Description string
	Reversible  bool
	Reversed    bool

	// Undo information
	UndoOperation string
	UndoDisk      string
	UndoIndex     string
	UndoSize      uint64
	UndoFSType    string

	// Original operation details
	Disk      string
	Index     string
	Size      uint64
	FSType    string
	OldSize   uint64
	OldFSType string
}

// OperationHistory manages the history of partition operations
type OperationHistory struct {
	entries    []*HistoryEntry
	nextID     int
	currentPos int // Position in history for undo/redo
	mu         sync.RWMutex
}

// NewOperationHistory creates a new operation history
func NewOperationHistory() *OperationHistory {
	return &OperationHistory{
		entries:    make([]*HistoryEntry, 0),
		nextID:     1,
		currentPos: -1,
	}
}

// RecordCreate records a partition creation operation
func (oh *OperationHistory) RecordCreate(disk, index string, size uint64, fsType string) {
	oh.mu.Lock()
	defer oh.mu.Unlock()

	// Remove any entries after current position (invalidate redo)
	if oh.currentPos < len(oh.entries)-1 {
		oh.entries = oh.entries[:oh.currentPos+1]
	}

	entry := &HistoryEntry{
		ID:            oh.nextID,
		Timestamp:     time.Now(),
		Operation:     "create",
		Description:   fmt.Sprintf("Created partition %s%s (%s, %.2f GB)", disk, index, fsType, float64(size)/(1024*1024*1024)),
		Reversible:    true,
		Reversed:      false,
		UndoOperation: "delete",
		UndoDisk:      disk,
		UndoIndex:     index,
		Disk:          disk,
		Index:         index,
		Size:          size,
		FSType:        fsType,
	}

	oh.entries = append(oh.entries, entry)
	oh.currentPos = len(oh.entries) - 1
	oh.nextID++
}

// RecordDelete records a partition deletion operation
func (oh *OperationHistory) RecordDelete(disk, index string, size uint64, fsType string) {
	oh.mu.Lock()
	defer oh.mu.Unlock()

	if oh.currentPos < len(oh.entries)-1 {
		oh.entries = oh.entries[:oh.currentPos+1]
	}

	entry := &HistoryEntry{
		ID:          oh.nextID,
		Timestamp:   time.Now(),
		Operation:   "delete",
		Description: fmt.Sprintf("Deleted partition %s%s (%s, %.2f GB)", disk, index, fsType, float64(size)/(1024*1024*1024)),
		Reversible:  false, // Cannot restore data
		Reversed:    false,
		Disk:        disk,
		Index:       index,
		Size:        size,
		FSType:      fsType,
	}

	oh.entries = append(oh.entries, entry)
	oh.currentPos = len(oh.entries) - 1
	oh.nextID++
}

// RecordFormat records a partition format operation
func (oh *OperationHistory) RecordFormat(partition, oldFSType, newFSType string) {
	oh.mu.Lock()
	defer oh.mu.Unlock()

	if oh.currentPos < len(oh.entries)-1 {
		oh.entries = oh.entries[:oh.currentPos+1]
	}

	entry := &HistoryEntry{
		ID:          oh.nextID,
		Timestamp:   time.Now(),
		Operation:   "format",
		Description: fmt.Sprintf("Formatted %s from %s to %s", partition, oldFSType, newFSType),
		Reversible:  false, // Cannot restore data
		Reversed:    false,
		Disk:        partition,
		FSType:      newFSType,
		OldFSType:   oldFSType,
	}

	oh.entries = append(oh.entries, entry)
	oh.currentPos = len(oh.entries) - 1
	oh.nextID++
}

// RecordResize records a partition resize operation
func (oh *OperationHistory) RecordResize(disk, index string, oldSize, newSize uint64) {
	oh.mu.Lock()
	defer oh.mu.Unlock()

	if oh.currentPos < len(oh.entries)-1 {
		oh.entries = oh.entries[:oh.currentPos+1]
	}

	entry := &HistoryEntry{
		ID:            oh.nextID,
		Timestamp:     time.Now(),
		Operation:     "resize",
		Description:   fmt.Sprintf("Resized %s%s from %.2f GB to %.2f GB", disk, index, float64(oldSize)/(1024*1024*1024), float64(newSize)/(1024*1024*1024)),
		Reversible:    true,
		Reversed:      false,
		UndoOperation: "resize",
		UndoDisk:      disk,
		UndoIndex:     index,
		UndoSize:      oldSize,
		Disk:          disk,
		Index:         index,
		Size:          newSize,
		OldSize:       oldSize,
	}

	oh.entries = append(oh.entries, entry)
	oh.currentPos = len(oh.entries) - 1
	oh.nextID++
}

// RecordCopy records a partition copy operation
func (oh *OperationHistory) RecordCopy(source, dest string, size uint64) {
	oh.mu.Lock()
	defer oh.mu.Unlock()

	if oh.currentPos < len(oh.entries)-1 {
		oh.entries = oh.entries[:oh.currentPos+1]
	}

	entry := &HistoryEntry{
		ID:          oh.nextID,
		Timestamp:   time.Now(),
		Operation:   "copy",
		Description: fmt.Sprintf("Copied %s to %s (%.2f GB)", source, dest, float64(size)/(1024*1024*1024)),
		Reversible:  false, // Cannot uncopy
		Reversed:    false,
		Disk:        source,
		Index:       dest,
		Size:        size,
	}

	oh.entries = append(oh.entries, entry)
	oh.currentPos = len(oh.entries) - 1
	oh.nextID++
}

// CanUndo returns true if there is an operation to undo
func (oh *OperationHistory) CanUndo() bool {
	oh.mu.RLock()
	defer oh.mu.RUnlock()

	if oh.currentPos < 0 {
		return false
	}

	// Check if current operation is reversible and not already reversed
	if oh.currentPos < len(oh.entries) {
		return oh.entries[oh.currentPos].Reversible && !oh.entries[oh.currentPos].Reversed
	}

	return false
}

// CanRedo returns true if there is an operation to redo
func (oh *OperationHistory) CanRedo() bool {
	oh.mu.RLock()
	defer oh.mu.RUnlock()

	// Can redo if we're not at the end and next operation was reversed
	if oh.currentPos < len(oh.entries)-1 {
		return oh.entries[oh.currentPos+1].Reversed
	}

	return false
}

// GetUndoOperation returns the operation to undo and moves position
func (oh *OperationHistory) GetUndoOperation() (*HistoryEntry, error) {
	oh.mu.Lock()
	defer oh.mu.Unlock()

	if oh.currentPos < 0 || oh.currentPos >= len(oh.entries) {
		return nil, fmt.Errorf("no operation to undo")
	}

	entry := oh.entries[oh.currentPos]
	if !entry.Reversible {
		return nil, fmt.Errorf("operation '%s' is not reversible", entry.Operation)
	}

	if entry.Reversed {
		return nil, fmt.Errorf("operation already reversed")
	}

	entry.Reversed = true
	oh.currentPos--

	return entry, nil
}

// GetRedoOperation returns the operation to redo and moves position
func (oh *OperationHistory) GetRedoOperation() (*HistoryEntry, error) {
	oh.mu.Lock()
	defer oh.mu.Unlock()

	if oh.currentPos >= len(oh.entries)-1 {
		return nil, fmt.Errorf("no operation to redo")
	}

	oh.currentPos++
	entry := oh.entries[oh.currentPos]

	if !entry.Reversed {
		return nil, fmt.Errorf("operation was not reversed")
	}

	entry.Reversed = false

	return entry, nil
}

// GetHistory returns all history entries
func (oh *OperationHistory) GetHistory() []*HistoryEntry {
	oh.mu.RLock()
	defer oh.mu.RUnlock()

	entries := make([]*HistoryEntry, len(oh.entries))
	copy(entries, oh.entries)
	return entries
}

// GetCurrentPosition returns the current position in history
func (oh *OperationHistory) GetCurrentPosition() int {
	oh.mu.RLock()
	defer oh.mu.RUnlock()

	return oh.currentPos
}

// RestorePosition restores the current position (for undo cancellation)
func (oh *OperationHistory) RestorePosition(pos int) {
	oh.mu.Lock()
	defer oh.mu.Unlock()

	if pos >= -1 && pos < len(oh.entries) {
		oh.currentPos = pos
	}
}

// RestoreReversedState restores the reversed state of an entry
func (oh *OperationHistory) RestoreReversedState(entryID int, reversed bool) {
	oh.mu.Lock()
	defer oh.mu.Unlock()

	for _, entry := range oh.entries {
		if entry.ID == entryID {
			entry.Reversed = reversed
			break
		}
	}
}

// Clear clears the entire history
func (oh *OperationHistory) Clear() {
	oh.mu.Lock()
	defer oh.mu.Unlock()

	oh.entries = make([]*HistoryEntry, 0)
	oh.currentPos = -1
}

// GetRecentEntries returns the most recent N entries
func (oh *OperationHistory) GetRecentEntries(count int) []*HistoryEntry {
	oh.mu.RLock()
	defer oh.mu.RUnlock()

	if count <= 0 {
		return []*HistoryEntry{}
	}

	start := len(oh.entries) - count
	if start < 0 {
		start = 0
	}

	entries := make([]*HistoryEntry, len(oh.entries)-start)
	copy(entries, oh.entries[start:])
	return entries
}

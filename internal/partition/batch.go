package partition

import (
	"fmt"
	"regexp"
	"sync"
)

// ParsePartitionName extracts disk name and partition index from a partition name
// Examples: ada0p1 -> (ada0, 1), ada0s1a -> (ada0, s1a)
func ParsePartitionName(partName string) (disk string, index string, err error) {
	// Match patterns like ada0p1, ada0s1, nvd0p2, etc.
	re := regexp.MustCompile(`^([a-z]+[0-9]+)([ps][0-9]+[a-z]?)$`)
	matches := re.FindStringSubmatch(partName)

	if len(matches) != 3 {
		return "", "", fmt.Errorf("invalid partition name format: %s", partName)
	}

	return matches[1], matches[2][1:], nil // Skip 'p' or 's' prefix
}

// OperationType represents the type of partition operation
type OperationType int

const (
	OpCreate OperationType = iota
	OpDelete
	OpFormat
	OpResize
	OpCopy
	OpMove
)

// String returns the string representation of the operation type
func (ot OperationType) String() string {
	switch ot {
	case OpCreate:
		return "Create"
	case OpDelete:
		return "Delete"
	case OpFormat:
		return "Format"
	case OpResize:
		return "Resize"
	case OpCopy:
		return "Copy"
	case OpMove:
		return "Move"
	default:
		return "Unknown"
	}
}

// BatchOperation represents a single queued operation
type BatchOperation struct {
	ID          int
	Type        OperationType
	Description string
	Status      string // "pending", "running", "completed", "failed"
	Error       string

	// Operation-specific parameters
	Disk           string
	Index          string
	Partition      string
	SourcePart     string
	DestPart       string
	SourceDisk     string
	SourceIndex    string
	DestDisk       string
	DestIndex      string
	FilesystemType string
	Size           uint64
}

// BatchQueue manages a queue of partition operations
type BatchQueue struct {
	operations []*BatchOperation
	nextID     int
	mu         sync.RWMutex
}

// NewBatchQueue creates a new batch queue
func NewBatchQueue() *BatchQueue {
	return &BatchQueue{
		operations: make([]*BatchOperation, 0),
		nextID:     1,
	}
}

// AddOperation adds a new operation to the queue
func (bq *BatchQueue) AddOperation(op *BatchOperation) int {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	op.ID = bq.nextID
	op.Status = "pending"
	bq.nextID++
	bq.operations = append(bq.operations, op)
	return op.ID
}

// RemoveOperation removes an operation from the queue by ID
func (bq *BatchQueue) RemoveOperation(id int) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	for i, op := range bq.operations {
		if op.ID == id {
			bq.operations = append(bq.operations[:i], bq.operations[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("operation with ID %d not found", id)
}

// MoveOperation moves an operation to a new position in the queue
func (bq *BatchQueue) MoveOperation(id int, newPosition int) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	var opIndex = -1
	for i, op := range bq.operations {
		if op.ID == id {
			opIndex = i
			break
		}
	}

	if opIndex == -1 {
		return fmt.Errorf("operation with ID %d not found", id)
	}

	if newPosition < 0 || newPosition >= len(bq.operations) {
		return fmt.Errorf("invalid position %d", newPosition)
	}

	// Remove from current position
	op := bq.operations[opIndex]
	bq.operations = append(bq.operations[:opIndex], bq.operations[opIndex+1:]...)

	// Insert at new position
	bq.operations = append(bq.operations[:newPosition], append([]*BatchOperation{op}, bq.operations[newPosition:]...)...)

	return nil
}

// GetOperations returns a copy of all operations
func (bq *BatchQueue) GetOperations() []*BatchOperation {
	bq.mu.RLock()
	defer bq.mu.RUnlock()

	ops := make([]*BatchOperation, len(bq.operations))
	copy(ops, bq.operations)
	return ops
}

// Clear removes all operations from the queue
func (bq *BatchQueue) Clear() {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	bq.operations = make([]*BatchOperation, 0)
}

// Count returns the number of operations in the queue
func (bq *BatchQueue) Count() int {
	bq.mu.RLock()
	defer bq.mu.RUnlock()

	return len(bq.operations)
}

// ExecuteAll executes all operations in the queue
func (bq *BatchQueue) ExecuteAll(stopOnError bool, progressCallback func(int, int, string)) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	total := len(bq.operations)
	if total == 0 {
		return fmt.Errorf("no operations to execute")
	}

	for i, op := range bq.operations {
		if op.Status == "completed" {
			continue
		}

		op.Status = "running"
		if progressCallback != nil {
			progressCallback(i+1, total, op.Description)
		}

		err := bq.executeOperation(op)
		if err != nil {
			op.Status = "failed"
			op.Error = err.Error()
			if stopOnError {
				return fmt.Errorf("operation %d failed: %v", op.ID, err)
			}
		} else {
			op.Status = "completed"
		}
	}

	return nil
}

// executeOperation executes a single operation
func (bq *BatchQueue) executeOperation(op *BatchOperation) error {
	switch op.Type {
	case OpCreate:
		return CreatePartition(op.Disk, op.Size, op.FilesystemType)

	case OpDelete:
		return DeletePartition(op.Disk, op.Index)

	case OpFormat:
		return FormatPartition(op.Partition, op.FilesystemType)

	case OpResize:
		return ResizePartition(op.Disk, op.Index, op.Size)

	case OpCopy:
		return CopyPartition(op.SourcePart, op.DestPart, nil)

	case OpMove:
		return MovePartition(op.SourceDisk, op.SourceIndex, op.DestDisk, op.DestIndex, nil)

	default:
		return fmt.Errorf("unknown operation type: %v", op.Type)
	}
}

// GetCompletedCount returns the number of completed operations
func (bq *BatchQueue) GetCompletedCount() int {
	bq.mu.RLock()
	defer bq.mu.RUnlock()

	count := 0
	for _, op := range bq.operations {
		if op.Status == "completed" {
			count++
		}
	}
	return count
}

// GetFailedCount returns the number of failed operations
func (bq *BatchQueue) GetFailedCount() int {
	bq.mu.RLock()
	defer bq.mu.RUnlock()

	count := 0
	for _, op := range bq.operations {
		if op.Status == "failed" {
			count++
		}
	}
	return count
}

// HasPendingOperations returns true if there are pending operations
func (bq *BatchQueue) HasPendingOperations() bool {
	bq.mu.RLock()
	defer bq.mu.RUnlock()

	for _, op := range bq.operations {
		if op.Status == "pending" {
			return true
		}
	}
	return false
}

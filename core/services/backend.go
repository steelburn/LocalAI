package services

import (
	"sync"

	"github.com/mudler/LocalAI/core/config"
	"github.com/rs/zerolog/log"
)

type BackendOpStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type BackendOp struct {
	Req       config.Backend
	Id        string
	BackendID string
	ConfigURL string
	Delete    bool
}

type BackendService struct {
	C      chan BackendOp
	status map[string]*BackendOpStatus
	mu     sync.RWMutex
}

func NewBackendService() *BackendService {
	return &BackendService{
		C:      make(chan BackendOp),
		status: make(map[string]*BackendOpStatus),
	}
}

func (s *BackendService) GetStatus(id string) *BackendOpStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status[id]
}

func (s *BackendService) GetAllStatus() map[string]*BackendOpStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *BackendService) Run() {
	for op := range s.C {
		s.mu.Lock()
		s.status[op.Id] = &BackendOpStatus{
			Status:  "processing",
			Message: "Processing backend operation",
		}
		s.mu.Unlock()

		var err error
		if op.Delete {
			err = s.deleteBackend(op)
		} else {
			err = s.installBackend(op)
		}

		s.mu.Lock()
		if err != nil {
			s.status[op.Id] = &BackendOpStatus{
				Status:  "error",
				Message: "Failed to process backend operation",
				Error:   err.Error(),
			}
		} else {
			s.status[op.Id] = &BackendOpStatus{
				Status:  "completed",
				Message: "Backend operation completed successfully",
			}
		}
		s.mu.Unlock()
	}
}

func (s *BackendService) installBackend(op BackendOp) error {
	log.Debug().Msgf("Installing backend %s", op.BackendID)
	// TODO: Implement backend installation logic
	return nil
}

func (s *BackendService) deleteBackend(op BackendOp) error {
	log.Debug().Msgf("Deleting backend %s", op.BackendID)
	// TODO: Implement backend deletion logic
	return nil
}

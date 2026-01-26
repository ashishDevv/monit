package monitor

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type UserService interface {
	IncrementMonitorCount(ctx context.Context, userID uuid.UUID) error
}

type Service struct {
	monitorRepo *Repository
	cache       Cache
	userSvc     UserService
	logger      *zerolog.Logger
}

func NewService(monitorRepo *Repository, cache Cache, userSvc UserService, logger *zerolog.Logger) *Service {
	return &Service{
		monitorRepo: monitorRepo,
		userSvc:     userSvc,
		cache:       cache,
		logger:      logger,
	}
}

func (s *Service) CreateMonitor(ctx context.Context, data CreateMonitorCmd) (uuid.UUID, error) {

	/*
		- 1 Increment Monitor count if possible
			- if monitors count is greater than than threshold, then it did not do any update, so no row effects.
		- 2 Now create a monitor
		- 3 schedule that monitor
			- if err occur after mutiple retries, then push it to background schedulers

		- one optimization -> do 1 and 2 in a transaction
	*/

	const op string = "service.monitor.create_monitor"

	err := s.userSvc.IncrementMonitorCount(ctx, data.UserID)
	if err != nil {
		return uuid.UUID{}, err
	}

	// now create monitor
	monitorID, err := s.monitorRepo.Create(ctx, data)
	if err != nil {
		return uuid.UUID{}, err
	}

	s.scheduleMonitor(ctx, monitorID, data.IntervalSec, op)

	return monitorID, nil
}

func (s *Service) GetMonitor(ctx context.Context, userID uuid.UUID, monitorID uuid.UUID) (Monitor, error) {
	// first check the redis cache
	// if found -> then return it
	// if not found
	// get from DB
	// store in cache
	// return it

	const op string = "service.monitor.get_monitor"

	m, exists := s.cache.GetMonitor(ctx, monitorID)
	if exists && m.UserID == userID {
		return m, nil
	}

	mDB, err := s.monitorRepo.Get(ctx, userID, monitorID)
	if err != nil { // err is already wraped in custom err
		return Monitor{}, err // so just return it
	}

	if err := s.cache.SetMonitor(ctx, mDB); err != nil {
		s.logger.Error().
			Str("op", op).
			Err(err).
			Msg("error in setting in cache")
	}

	return mDB, nil
}

func (s *Service) LoadMonitor(ctx context.Context, monitorID uuid.UUID) (Monitor, error) {
	// first check the redis cache
	// if found -> then return it
	// if not found
	// get from DB
	// store in cache
	// return it

	const op string = "service.monitor.load_monitor"

	m, exists := s.cache.GetMonitor(ctx, monitorID)
	if exists {
		return m, nil
	}

	mDB, err := s.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		return Monitor{}, err
	}

	if err := s.cache.SetMonitor(ctx, mDB); err != nil {
		s.logger.Error().
			Str("op", op).
			Err(err).
			Msg("error in setting in cache")
	}

	return mDB, nil
}

func (s *Service) GetAllMonitors(ctx context.Context, userID uuid.UUID, limit int32, offset int32) ([]Monitor, error) {
	m, err := s.monitorRepo.GetAll(ctx, userID, limit, offset)
	if err != nil {
		return []Monitor{}, err
	}
	return m, nil
}

func (s *Service) UpdateMonitorStatus(ctx context.Context, userID, monitorID uuid.UUID, enable bool) (bool, error) {

	/*
		check current status of monitor
		if user given command is same as the current state of monitor -> do nothing and return
		if not then
		if user gives disable command and currently monitor is enabled
			then we do monitor disable process
				first make change monitor status in db to enable = false
				now delete the cached monitor from redis
				delete the sechedule entry from redis set
				delete the running incident from redis
				delete the entry of status from redis
				means overall clear all cached related to that monitorID

		if user gives enable command and currently monitor is disabled
			then we do monitor enable process
				we just schedule the monitor
	*/

	const op = "service.monitor.update_status"

	// 1. Load monitor (auth enforced)
	m, err := s.monitorRepo.Get(ctx, userID, monitorID)
	if err != nil {
		return false, err
	}

	// 2. Idempotent behavior
	if m.Enabled == enable {
		return true, nil
	}

	// 3. Persist desired state
	if err := s.monitorRepo.SetEnabled(ctx, userID, monitorID, enable); err != nil {
		return false, err
	}

	// 4. Side effects (best effort)
	if enable {
		s.scheduleMonitor(ctx, m.ID, m.IntervalSec, op)
	} else {
		s.disableMonitor(ctx, monitorID)
	}

	return true, nil
}

func (s *Service) scheduleMonitor(ctx context.Context, mID uuid.UUID, intervalSec int32, op string) {

	nextRun := time.Now().Add(time.Duration(intervalSec) * time.Second)

	if err := s.cache.Schedule(ctx, mID.String(), nextRun); err != nil {
		s.logger.Error().
			Str("op", op).
			Err(err).
			Msg("Error in scheduling monitor, after multiple retries, will retry asynchronously")
		// enqueue retry job
		// Now put these monitors in the a another channel, where
		// workers schedule them in background
		// so that the state remain consistent
	}
}

func (s *Service) disableMonitor(ctx context.Context, monitorID uuid.UUID) {
	// delete cached monitor (if any)
	_ = s.cache.DelMonitor(ctx, monitorID)
	// delete scheduled entry
	_ = s.cache.DelSchedule(ctx, monitorID.String())
	// delete incident (if any)
	_ = s.cache.ClearIncident(ctx, monitorID)
	// delete status entry
	_ = s.cache.DelStatus(ctx, monitorID)
}

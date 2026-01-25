package monitor

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type UserService interface {
	GetMonitorQuota(context.Context, uuid.UUID) (int32, error)
	IncrementMonitorCount(context.Context, uuid.UUID) error
}

type Service struct {
	monitorRepo *Repository
	cache       Cache
	userSvc     UserService
}

func NewService(monitorRepo *Repository, cache Cache, userSvc UserService) *Service {
	return &Service{
		monitorRepo: monitorRepo,
		userSvc:     userSvc,
		cache:       cache,
	}
}

func (s *Service) CreateMonitor(ctx context.Context, data CreateMonitorCmd) (uuid.UUID, error) {

	// Step:
	// 	Check number of current monitors in user quota
	// 	if exceed then reject the req
	//  if not then create new MonitorRecord
	// now increment monitor_count by 1
	//  now schedule it on redisstore
	//  return monitor id

	quota, err := s.userSvc.GetMonitorQuota(ctx, data.UserID)
	if err != nil {
		return uuid.UUID{}, err
	}
	if quota >= 10 {
		return uuid.UUID{}, err
	}

	monitorID, err := s.monitorRepo.Create(ctx, data)
	if err != nil {
		return uuid.UUID{}, err
	}
	if err := s.userSvc.IncrementMonitorCount(ctx, data.UserID); err != nil {
		return uuid.UUID{}, err
	}

	nextRun := time.Now().Add(time.Duration(data.IntervalSec) * time.Second)
	if err := s.cache.Schedule(ctx, monitorID.String(), nextRun); err != nil {
		return uuid.UUID{}, err
	}

	return monitorID, nil
}

func (s *Service) GetMonitor(ctx context.Context, userID uuid.UUID, monitorID uuid.UUID) (Monitor, error) {
	// first check the redis cache
	// if found -> then return it
	// if not found
	// get from DB
	// store in cache
	// return it

	m, exists := s.cache.GetMonitor(ctx, monitorID)
	if exists {
		if m.UserID == userID {
			return m, nil
		}
		return Monitor{}, errors.New("Unauthorised")
	}

	mDB, err := s.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		return Monitor{}, err
	}
	if mDB.UserID != userID {
		return Monitor{}, errors.New("Unauthorised")
	}
	_ = s.cache.SetMonitor(ctx, mDB)

	return mDB, nil
}

func (s *Service) LoadMonitor(ctx context.Context, monitorID uuid.UUID) (Monitor, error) {
	// first check the redis cache
	// if found -> then return it
	// if not found
	// get from DB
	// store in cache
	// return it

	m, exists := s.cache.GetMonitor(ctx, monitorID)
	if exists {
		return m, nil
	}

	mDB, err := s.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		return Monitor{}, err
	}
	_ = s.cache.SetMonitor(ctx, mDB)

	return mDB, nil
}

func (s *Service) GetAllMonitors(ctx context.Context, userID uuid.UUID, limit int32, offset int32) ([]Monitor, error) {
	m, err := s.monitorRepo.GetAll(ctx, userID, limit, offset)
	if err != nil {
		return nil, nil
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

	m, err := s.monitorRepo.Get(ctx, userID, monitorID)
	if err != nil {
		return false, err // error occur
	}
	if m.Enabled == enable {
		return false, errors.New("same state") // state is same
	}
	if enable == true && m.Enabled == false {
		// we do monitor enable process
		nextRun := time.Now().Add(time.Duration(m.IntervalSec) * time.Second)
		if err := s.cache.Schedule(ctx, monitorID.String(), nextRun); err != nil {
			return false, errors.New("Internal error")
		}
		return true, nil
	}
	if enable == false && m.Enabled == true {
		// we do monitor disable process
		_, err := s.monitorRepo.EnableDisableMonitor(ctx, userID, monitorID, false)
		if err != nil {
			return false, err   // db error
		}

		// delete cached monitor (if any)
		_ = s.cache.DelMonitor(ctx, monitorID)
		// delete scheduled entry
		_ = s.cache.DelSchedule(ctx, monitorID.String())
		// delete incident (if any)
		_ = s.cache.ClearIncident(ctx, monitorID)
		// delete status entry
		_ = s.cache.DelStatus(ctx, monitorID)
	}
	return true, nil
}

package id_generator

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/starslipay/trade_id_mgr/model/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type IdSegmentCache struct {
	mu              sync.Mutex
	curID           int64
	segmentStart    int64
	segmentEnd      int64
	minID           int64
	maxGlobalID     int64
	stepSize        int
	sceneID         int64
	nextCurID       int64
	nextSegmentEnd  int64
	nextMinID       int64
	nextMaxGlobalID int64
	nextStepSize    int
	prefetching     atomic.Bool
}

type IDGenerator struct {
	sceneMap sync.Map
	db       mysql.TIdSegmentModel
	conn     sqlx.SqlConn
}

func MustNewIDGenerator(db mysql.TIdSegmentModel, conn sqlx.SqlConn, SceneID []int64) *IDGenerator {
	gen := &IDGenerator{
		db:       db,
		conn:     conn,
		sceneMap: sync.Map{},
	}

	for _, sceneID := range SceneID {
		gen.sceneMap.Store(sceneID, &IdSegmentCache{
			sceneID: sceneID,
		})
	}

	if err := gen.loadAllSceneSegments(context.Background()); err != nil {
		panic(err)
	}

	return gen
}

func (g *IDGenerator) fetchSegment(ctx context.Context, sceneID int64) (curID, segmentEnd, minID, maxGlobalID int64, stepSize int, err error) {
	err = g.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		txDB := g.db.WithSession(session)

		segment, err := txDB.FindOneForUpdate(ctx, sceneID)
		if err != nil {
			return err
		}

		minID = segment.MinId
		maxGlobalID = segment.MaxId
		stepSize = int(segment.StepSize)

		newMaxAllocated := segment.MaxAllocatedId + int64(stepSize)
		if newMaxAllocated > maxGlobalID {
			curID = minID
			segmentEnd = minID + int64(stepSize) - 1
			newMaxAllocated = segmentEnd
		} else {
			curID = segment.MaxAllocatedId + 1
			segmentEnd = newMaxAllocated
		}

		segment.MaxAllocatedId = newMaxAllocated
		return txDB.Update(ctx, segment)
	})
	return
}

func (g *IDGenerator) loadAllSceneSegments(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	g.sceneMap.Range(func(key, value interface{}) bool {
		cache := value.(*IdSegmentCache)

		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := g.loadSegmentToCache(ctx, cache, false); err != nil {
				select {
				case errChan <- err:
				default:
				}
			}
		}()

		return true
	})

	wg.Wait()
	close(errChan)

	return <-errChan
}

func (g *IDGenerator) GetID(ctx context.Context, sceneID int64) (int64, error) {
	value, ok := g.sceneMap.Load(sceneID)
	if !ok {
		return 0, ErrSceneNotFound
	}

	cache := value.(*IdSegmentCache)
	cache.mu.Lock()

	if cache.curID > cache.segmentEnd {
		if cache.nextCurID > 0 && cache.nextSegmentEnd > 0 {
			cache.curID = cache.nextCurID
			cache.segmentStart = cache.nextCurID
			cache.segmentEnd = cache.nextSegmentEnd
			cache.minID = cache.nextMinID
			cache.maxGlobalID = cache.nextMaxGlobalID
			cache.stepSize = cache.nextStepSize
			cache.nextCurID = 0
			cache.nextSegmentEnd = 0
			cache.nextMinID = 0
			cache.nextMaxGlobalID = 0
			cache.nextStepSize = 0
			cache.prefetching.Store(false)
		} else {
			cache.mu.Unlock()

			if err := g.loadSegmentToCache(ctx, cache, false); err != nil {
				return 0, err
			}

			cache.mu.Lock()
		}
	}

	id := cache.curID
	cache.curID++

	threshold := cache.segmentStart + (cache.segmentEnd-cache.segmentStart)*4/5
	if cache.curID > threshold && !cache.prefetching.Load() {
		cache.prefetching.Store(true)
		go g.asyncPrefetch(ctx, sceneID, cache)
	}

	cache.mu.Unlock()
	return id, nil
}

func (g *IDGenerator) loadSegmentToCache(ctx context.Context, cache *IdSegmentCache, isNext bool) error {
	curID, segmentEnd, minID, maxGlobalID, stepSize, err := g.fetchSegment(ctx, cache.sceneID)
	if err != nil {
		if isNext {
			cache.prefetching.Store(false)
		}
		return err
	}

	cache.mu.Lock()
	if isNext {
		cache.nextCurID = curID
		cache.nextSegmentEnd = segmentEnd
		cache.nextMinID = minID
		cache.nextMaxGlobalID = maxGlobalID
		cache.nextStepSize = stepSize
	} else {
		cache.curID = curID
		cache.segmentStart = curID
		cache.segmentEnd = segmentEnd
		cache.minID = minID
		cache.maxGlobalID = maxGlobalID
		cache.stepSize = stepSize
	}
	cache.mu.Unlock()
	return nil
}

func (g *IDGenerator) asyncPrefetch(ctx context.Context, sceneID int64, cache *IdSegmentCache) {
	_ = g.loadSegmentToCache(ctx, cache, true)
}

var ErrSceneNotFound = &SceneNotFoundError{}

type SceneNotFoundError struct{}

func (e *SceneNotFoundError) Error() string {
	return "scene not found"
}

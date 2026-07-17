package id_generator

import (
	"context"
	"log"
	"sync"

	"github.com/starslipay/trade_id_mgr/model/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type IdSegmentCache struct {
	curMu        sync.Mutex
	curID        int64
	segmentStart int64
	segmentEnd   int64
	minID        int64
	maxGlobalID  int64
	stepSize     int

	nextMu          sync.Mutex
	nextCurID       int64
	nextSegmentEnd  int64
	nextMinID       int64
	nextMaxGlobalID int64
	nextStepSize    int

	loadMu  sync.Mutex
	sceneID int64
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

			cache.loadMu.Lock()
			defer cache.loadMu.Unlock()

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

	cache.curMu.Lock()
	if cache.curID > cache.segmentEnd { // 当前缓存已耗尽
		cache.curMu.Unlock()

		cache.curMu.Lock()
		cache.nextMu.Lock()

		if cache.curID > cache.segmentEnd {
			if cache.nextCurID > 0 && cache.nextSegmentEnd > 0 { // 判断nextBuf是否为空，若为空则切换缓存，否则返回错误
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
				log.Printf("[IDGenerator] scene=%d, 切换缓存: curBuf=[%d,%d], nextBuf清空", sceneID, cache.curID, cache.segmentEnd)
			} else {
				cache.nextMu.Unlock()
				cache.curMu.Unlock()
				return 0, ErrSegmentExhausted
			}
		}

		cache.nextMu.Unlock()
	}

	id := cache.curID
	cache.curID++

	threshold := cache.segmentStart + (cache.segmentEnd-cache.segmentStart)*4/5
	remaining := cache.segmentEnd - cache.curID + 1
	total := cache.segmentEnd - cache.segmentStart + 1
	usagePercent := float64(total-remaining) / float64(total) * 100

	cache.nextMu.Lock()
	nextCurID := cache.nextCurID
	nextSegmentEnd := cache.nextSegmentEnd
	cache.nextMu.Unlock()

	log.Printf("[IDGenerator] scene=%d, GetID=%d, segment=[%d,%d], threshold=%d, remaining=%d/%d (%.2f%%), nextCache=[%d,%d]",
		sceneID, id, cache.segmentStart, cache.segmentEnd, threshold, remaining, total, usagePercent, nextCurID, nextSegmentEnd)

	needPrefetch := cache.curID > threshold && nextCurID == 0
	if needPrefetch {
		log.Printf("[IDGenerator] scene=%d, 触发异步预取: curID=%d > threshold=%d, nextBuf为空", sceneID, cache.curID, threshold)
	}

	cache.curMu.Unlock()

	if needPrefetch {
		go g.asyncPrefetch(context.Background(), sceneID, cache)
	}

	return id, nil
}

func (g *IDGenerator) loadSegmentToCache(ctx context.Context, cache *IdSegmentCache, isNext bool) error {
	log.Printf("[IDGenerator] loadSegmentToCache start: scene=%d, isNext=%v", cache.sceneID, isNext)

	curID, segmentEnd, minID, maxGlobalID, stepSize, err := g.fetchSegment(ctx, cache.sceneID)
	if err != nil {
		log.Printf("[IDGenerator] loadSegmentToCache failed: scene=%d, fetchSegment error=%v", cache.sceneID, err)
		return err
	}

	if isNext {
		cache.nextMu.Lock()
		cache.nextCurID = curID
		cache.nextSegmentEnd = segmentEnd
		cache.nextMinID = minID
		cache.nextMaxGlobalID = maxGlobalID
		cache.nextStepSize = stepSize
		log.Printf("[IDGenerator] loadSegmentToCache success(next): scene=%d, nextSegment=[%d,%d]", cache.sceneID, curID, segmentEnd)
		cache.nextMu.Unlock()
	} else {
		cache.curMu.Lock()
		cache.curID = curID
		cache.segmentStart = curID
		cache.segmentEnd = segmentEnd
		cache.minID = minID
		cache.maxGlobalID = maxGlobalID
		cache.stepSize = stepSize
		log.Printf("[IDGenerator] loadSegmentToCache success(current): scene=%d, segment=[%d,%d]", cache.sceneID, curID, segmentEnd)
		cache.curMu.Unlock()
	}
	return nil
}

func (g *IDGenerator) asyncPrefetch(ctx context.Context, sceneID int64, cache *IdSegmentCache) {
	log.Printf("[IDGenerator] asyncPrefetch start: scene=%d", sceneID)

	cache.loadMu.Lock()
	defer cache.loadMu.Unlock()

	cache.nextMu.Lock()
	nextCurID := cache.nextCurID
	cache.nextMu.Unlock()

	if nextCurID > 0 {
		log.Printf("[IDGenerator] asyncPrefetch skip: scene=%d, nextBuf already has data=[%d,%d]", sceneID, nextCurID, cache.nextSegmentEnd)
		return
	}

	if err := g.loadSegmentToCache(ctx, cache, true); err != nil {
		log.Printf("[IDGenerator] asyncPrefetch failed: scene=%d, error=%v", sceneID, err)
	} else {
		log.Printf("[IDGenerator] asyncPrefetch completed: scene=%d", sceneID)
	}
}

var ErrSceneNotFound = &SceneNotFoundError{}
var ErrSegmentExhausted = &SegmentExhaustedError{}

type SceneNotFoundError struct{}

func (e *SceneNotFoundError) Error() string {
	return "scene not found"
}

type SegmentExhaustedError struct{}

func (e *SegmentExhaustedError) Error() string {
	return "segment exhausted"
}

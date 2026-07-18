package id_generator

import (
	"context"
	"log"
	"sync"
	"sync/atomic"

	"github.com/starslipay/trade_id_mgr/model/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type IdSegmentCache struct {
	curID        int64
	segmentStart int64
	segmentEnd   int64
}

// ID号段双缓存，用于存储当前正在使用的缓存和备用缓存
type IdSegmentDoubleCache struct {
	sceneID       int64
	mu            sync.Mutex
	activeCache   IdSegmentCache // 活跃缓存，当前正在使用的缓存
	standbyCache  IdSegmentCache // 备用缓存，用于存储下一个缓存
	isPreFetching atomic.Bool    //是否正在预取备用缓存数据，为什么用原子变量：保证无锁情况下内存可见性
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
		gen.sceneMap.Store(sceneID, &IdSegmentDoubleCache{
			sceneID: sceneID,
		})
	}

	if err := gen.loadAllSceneSegments(context.Background()); err != nil {
		panic(err)
	}

	return gen
}

func (g *IDGenerator) fetchSegmentFromDB(ctx context.Context, sceneID int64) (segmentStart, segmentEnd int64, err error) {
	err = g.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		txDB := g.db.WithSession(session)

		segment, err := txDB.FindOneForUpdate(ctx, sceneID)
		if err != nil {
			return err
		}

		minID := segment.MinId
		maxID := segment.MaxId
		stepSize := int(segment.StepSize)

		newMaxAllocated := segment.MaxAllocatedId + int64(stepSize)
		if newMaxAllocated > maxID {
			segmentStart = minID
			segmentEnd = minID + int64(stepSize) - 1
			newMaxAllocated = segmentEnd
		} else {
			segmentStart = segment.MaxAllocatedId + 1
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

	// 遍历map，并发加载所有场景的ID号段
	g.sceneMap.Range(func(key, value interface{}) bool {
		doubleCache := value.(*IdSegmentDoubleCache)

		wg.Add(1)
		go func() {
			defer wg.Done()

			segmentStart, segmentEnd, err := g.fetchSegmentFromDB(ctx, doubleCache.sceneID)
			if err != nil {
				log.Printf("scene %d init seg failed: %v", doubleCache.sceneID, err)

				// 非阻塞写入错误通道, 如果通道中有数据, 则直接跳过写入
				select {
				case errChan <- err:
				default:
				}
				// 失败直接return，不更新缓存
				return
			}

			// 只有数据库拉取成功才更新号段缓存
			doubleCache.mu.Lock()
			doubleCache.activeCache.curID = segmentStart
			doubleCache.activeCache.segmentStart = segmentStart
			doubleCache.activeCache.segmentEnd = segmentEnd
			doubleCache.mu.Unlock()

			log.Printf("[IDGenerator] scene=%d, 初始化完成: segment=[%d,%d]", doubleCache.sceneID, segmentStart, segmentEnd)
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

	doubleCache := value.(*IdSegmentDoubleCache)

	doubleCache.mu.Lock()

	// cache.curID > cache.segmentEnd时说明当前缓存用完了
	if doubleCache.activeCache.curID > doubleCache.activeCache.segmentEnd {
		// cache.standbyCache.segmentStart > 0 && cache.standbyCache.segmentEnd > 0 说明备用缓存有数据
		if doubleCache.standbyCache.segmentStart > 0 && doubleCache.standbyCache.segmentEnd > 0 {
			// 将备用缓存的数据切换到当前缓存
			doubleCache.activeCache.curID = doubleCache.standbyCache.segmentStart
			doubleCache.activeCache.segmentStart = doubleCache.standbyCache.segmentStart
			doubleCache.activeCache.segmentEnd = doubleCache.standbyCache.segmentEnd
			doubleCache.standbyCache.curID = 0
			doubleCache.standbyCache.segmentStart = 0
			doubleCache.standbyCache.segmentEnd = 0
			log.Printf("[IDGenerator] scene=%d, 切换缓存: curBuf=[%d,%d]", sceneID, doubleCache.activeCache.curID, doubleCache.activeCache.segmentEnd)
		} else {
			doubleCache.mu.Unlock()
			return 0, ErrSegmentExhausted
		}
	}

	id := doubleCache.activeCache.curID
	doubleCache.activeCache.curID++

	// 计算需要触发异步预取的ID阈值 80%
	// 80% 是根据经验，根据实际场景调整
	threshold := doubleCache.activeCache.segmentStart + (doubleCache.activeCache.segmentEnd-doubleCache.activeCache.segmentStart)*4/5
	// 计算当前缓存已使用个数
	remaining := doubleCache.activeCache.segmentEnd - doubleCache.activeCache.curID + 1
	// 计算当前缓存总个数
	total := doubleCache.activeCache.segmentEnd - doubleCache.activeCache.segmentStart + 1
	// 计算当前缓存已使用率
	usagePercent := float64(total-remaining) / float64(total) * 100

	// 自带 Load 内存屏障（acquire 语义），强制本次读取绕过 CPU 私有缓存，直接从主内存拿最新数据
	// 禁止跨屏障指令乱序执行, 读取到
	isPreFetching := doubleCache.isPreFetching.Load()
	log.Printf("[IDGenerator] scene=%d, GetID=%d, segment=[%d,%d], threshold=%d, remaining=%d/%d (%.2f%%), standbyCache=[%d,%d], isStandbyHasData=%v",
		sceneID, id, doubleCache.activeCache.segmentStart, doubleCache.activeCache.segmentEnd, threshold, remaining, total, usagePercent, doubleCache.standbyCache.segmentStart, doubleCache.activeCache.segmentEnd, isPreFetching)
	// 当前缓存使用率超过阈值，且备用缓存无数据，且当前没有正在取备用缓存数据，需触发异步预取
	isNeedPreFetch := doubleCache.activeCache.curID > threshold && doubleCache.standbyCache.segmentStart == 0 && !isPreFetching
	if isNeedPreFetch {
		// 触发异步预取前，提前设置正在正在取备用缓存数据, 防止并发情况下多个协程同时触发异步预取
		doubleCache.isPreFetching.Store(true)
		log.Printf("[IDGenerator] scene=%d, 触发异步预取: curID=%d > threshold=%d", sceneID, doubleCache.activeCache.curID, threshold)
	}
	doubleCache.mu.Unlock()

	// 并发情况下只有一个协程会执行异步预取
	if isNeedPreFetch {
		// 使用独立的ctx，避免主协程取消时，异步预取协程也取消执行
		go g.asyncPrefetch(context.Background(), sceneID, doubleCache)
	}

	return id, nil
}

func (g *IDGenerator) asyncPrefetch(ctx context.Context, sceneID int64, doubleCache *IdSegmentDoubleCache) {
	log.Printf("[IDGenerator] asyncPrefetch start: scene=%d", sceneID)

	segmentStart, segmentEnd, err := g.fetchSegmentFromDB(ctx, sceneID)
	if err != nil {
		log.Printf("[IDGenerator] asyncPrefetch failed: scene=%d, error=%v", sceneID, err)
		// 失败后，取消正在取备用缓存数据的标志
		doubleCache.isPreFetching.Store(false)
		return
	}

	// 只有数据库拉取成功才更新号段缓存
	doubleCache.mu.Lock()
	doubleCache.standbyCache.curID = segmentStart
	doubleCache.standbyCache.segmentStart = segmentStart
	doubleCache.standbyCache.segmentEnd = segmentEnd
	doubleCache.mu.Unlock()

	log.Printf("[IDGenerator] asyncPrefetch completed: scene=%d, nextBuf=[%d,%d]", sceneID, segmentStart, segmentEnd)
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

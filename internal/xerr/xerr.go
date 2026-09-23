package xerr

// 错误码  10000 0000 ~~99999 9999
// 模块id  80000
// 错误码 = 模块id + 业务错误码
var (
	ModuleId = int64(455903)
)

var (
	// 系统错误 0000-0999
	ErrCodeDB             = int64(455903000)
	ErrCodeServerInternal = int64(455903001)

	// 业务错误码 1000-1999
	ErrCodeSceneNotFound    = int64(455903100)
	ErrCodeSegmentExhausted = int64(455903101)
)

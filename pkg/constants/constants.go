package constants

// Log level constants for structured logging, starting from 1
// 0 is default if no level is provided
// Guidelines: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md#what-method-to-use
const (
	LvlDefault = iota + 1 // 1 - General configuration, routine logs
	LvlInfo               // 2 - Steady-state operations, HTTP requests, system state changes
	LvlEvent              // 3 - Extended changes, additional system details
	LvlDebug              // 4 - Debug-level logs, tricky logic areas
	LvlTrace              // 5 - Trace-level logs, detailed troubleshooting context
)

// Action constants for error translation context
// These align with the driver operations and underlying API calls
const (
	ActionCreateBucket       = "CreateBucket"
	ActionDeleteBucket       = "DeleteBucket"
	ActionGrantBucketAccess  = "GrantBucketAccess"
	ActionRevokeBucketAccess = "RevokeBucketAccess"
)

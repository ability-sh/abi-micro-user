package srv

const (
	SERVICE_LRUCACHE = "lrucache"
	SERVICE_REDIS    = "redis"
	SERVICE_OSS      = "oss"
	SERVICE_MONGODB  = "mongodb"
	SERVICE_USER     = "uv-user"
)

const (
	ERRNO_OK              = 200
	ERRNO_NOT_FOUND       = 404
	ERRNO_INTERNAL_SERVER = 500
	ERRNO_INPUT_DATA      = 400
	ERRNO_INDEX_VALUE     = 600
	ERRNO_LOGIN           = 601
)

package orm

var (
	config = MysqlDBConfig{
		User:    "root",
		Pass:    "123456",
		Url:     "127.0.0.1:3306",
		Schema:  "test",
		MaxConn: 100,
		MaxIdle: 10,
	}
	dbpool, err = config.NewDBPool()
)

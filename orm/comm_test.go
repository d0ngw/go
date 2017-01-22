package orm

var (
	config = MysqlDBConfig{
		"root",
		"123456",
		"127.0.0.1:3306",
		"test",
		100,
		10}
	dbpool, err = config.CreateDBPool()
)

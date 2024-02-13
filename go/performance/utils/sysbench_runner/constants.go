package sysbench_runner

const (
	Dolt     ServerType = "dolt"
	Doltgres ServerType = "doltgres"
	Postgres ServerType = "postgres"
	MySql    ServerType = "mysql"

	CsvFormat  = "csv"
	JsonFormat = "json"

	CsvExt  = ".csv"
	JsonExt = ".json"

	defaultHost         = "127.0.0.1"
	defaultDoltPort     = 3306
	defaultMysqlPort    = defaultDoltPort
	defaultDoltgresPort = 5432
	defaultPostgresPort = defaultDoltgresPort

	defaultMysqlSocket = "/var/run/mysqld/mysqld.sock"

	tcpProtocol  = "tcp"
	unixProtocol = "unix"

	sysbenchUsername        = "sysbench"
	sysbenchUserLocal       = "'sysbench'@'localhost'"
	sysbenchPassLocal       = "sysbenchpass"
	sysbenchDbPsModeFlag    = "--db-ps-mode"
	sysbenchDbPsModeDisable = "disable"
	sysbenchRandTypeFlag    = "--rand-type"
	sysbenchRandTypeUniform = "uniform"

	doltSqlServerCommand = "sql-server"

	userFlag                    = "--user"
	hostFlag                    = "--host"
	portFlag                    = "--port"
	skipBinLogFlag              = "--skip-log-bin"
	profileFlag                 = "--prof"
	profilePathFlag             = "--prof-path"
	cpuProfile                  = "cpu"
	doltgresDataDirFlag         = "--data-dir"
	MysqlDataDirFlag            = "--datadir"
	MysqlInitializeInsecureFlag = "--initialize-insecure"
	cpuProfileFilename          = "cpu.pprof"

	sysbenchOltpReadOnlyTestName       = "oltp_read_only"
	sysbenchOltpInsertTestName         = "oltp_insert"
	sysbenchBulkInsertTestName         = "bulk_insert"
	sysbenchOltpPointSelectTestName    = "oltp_point_select"
	sysbenchSelectRandomPointsTestName = "select_random_points"
	sysbenchSelectRandomRangesTestName = "select_random_ranges"
	sysbenchOltpWriteOnlyTestName      = "oltp_write_only"
	sysbenchOltpReadWriteTestName      = "oltp_read_write"
	sysbenchOltpUpdateIndexTestName    = "oltp_update_index"
	sysbenchOltpUpdateNonIndexTestName = "oltp_update_non_index"

	sysbenchCoveringIndexScanLuaTestName = "covering_index_scan.lua"
	sysbenchGroupByScanLuaTestName       = "groupby_scan.lua"
	sysbenchIndexJoinLuaTestName         = "index_join.lua"
	sysbenchIndexJoinScanLuaTestName     = "index_join_scan.lua"
	sysbenchIndexScanLuaTestName         = "index_scan.lua"
	sysbenchOltpDeleteInsertLuaTestName  = "oltp_delete_insert.lua"
	sysbenchTableScanLuaTestName         = "table_scan.lua"
	sysbenchTypesDeleteInsertLuaTestName = "types_delete_insert.lua"
	sysbenchTypesTableScanLuaTestName    = "types_table_scan.lua"

	sysbenchCoveringIndexScanPostgresLuaTestName = "covering_index_scan_postgres.lua"
	sysbenchGroupByScanPostgresLuaTestName       = "groupby_scan_postgres.lua"
	sysbenchIndexJoinPostgresLuaTestName         = "index_join_postgres.lua"
	sysbenchIndexJoinScanPostgresLuaTestName     = "index_join_scan_postgres.lua"
	sysbenchIndexScanPostgresLuaTestName         = "index_scan_postgres.lua"
	sysbenchOltpDeleteInsertPostgresLuaTestName  = "oltp_delete_insert_postgres.lua"
	sysbenchTableScanPostgresLuaTestName         = "table_scan_postgres.lua"
	sysbenchTypesDeleteInsertPostgresLuaTestName = "types_delete_insert_postgres.lua"
	sysbenchTypesTableScanPostgresLuaTestName    = "types_table_scan_postgres.lua"

	doltConfigUsernameKey = "user.name"
	doltConfigEmailKey    = "user.email"
	doltBenchmarkUser     = "benchmark"
	doltBenchmarkEmail    = "benchmark@dolthub.com"
	doltConfigCommand     = "config"
	doltConfigGlobalFlag  = "--global"
	doltConfigGetFlag     = "--get"
	doltConfigAddFlag     = "--add"
	doltCloneCommand      = "clone"
	doltVersionCommand    = "version"
	doltInitCommand       = "init"
	dbName                = "test"
	bigEmptyRepo          = "max-hoffman/big-empty"
	nbfEnvVar             = "DOLT_DEFAULT_BIN_FORMAT"

	expectedServerKilledErrorMessage     = "signal: killed"
	expectedServerTerminatedErrorMessage = "signal: terminated"

	sysbenchCommand       = "sysbench"
	luaPathEnvVarTemplate = "LUA_PATH=%s"
	luaPath               = "?.lua"

	tpccDbName              = "sbt"
	tpccScaleFactorTemplate = "tpcc-scale-factor-%d"
)

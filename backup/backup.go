package backup

import (
	"flag"
	"fmt"

	"github.com/greenplum-db/gpbackup/utils"
)

var (
	connection *utils.DBConn
	logger     *utils.Logger
)

var ( // Command-line flags
	dbname  = flag.String("dbname", "", "The database to be backed up")
	debug   = flag.Bool("debug", false, "Print verbose and debug log messages")
	dumpDir = flag.String("dumpdir", "", "The directory to which all dump files will be written")
	quiet   = flag.Bool("quiet", false, "Suppress non-warning, non-error log messages")
	verbose = flag.Bool("verbose", false, "Print verbose log messages")
)

// This function handles setup that can be done before parsing flags.
func DoInit() {
	SetLogger(utils.InitializeLogging("gpbackup", ""))
}

func SetLogger(log *utils.Logger) {
	logger = log
}

/*
* This function handles argument parsing and validation, e.g. checking that a passed filename exists.
* It should only validate; initialization with any sort of side effects should go in DoInit or DoSetup.
 */
func DoValidation() {
	flag.Parse()
	utils.CheckExclusiveFlags("debug", "quiet", "verbose")
}

// This function handles setup that must be done after parsing flags.
func DoSetup() {
	if *quiet {
		logger.SetVerbosity(utils.LOGERROR)
	} else if *debug {
		logger.SetVerbosity(utils.LOGDEBUG)
	} else if *verbose {
		logger.SetVerbosity(utils.LOGVERBOSE)
	}
	connection = utils.NewDBConn(*dbname)
	connection.Connect()
	connection.Exec("SET application_name TO 'gpbackup'")

	utils.SetDumpTimestamp("")

	if *dumpDir != "" {
		utils.BaseDumpDir = *dumpDir
	}
	logger.Verbose("Creating dump directories")
	segConfig := utils.GetSegmentConfiguration(connection)
	utils.SetupSegmentConfiguration(segConfig)
	utils.CreateDumpDirs()
}

func DoBackup() {
	logger.Info("Dump Key = %s", utils.DumpTimestamp)
	logger.Info("Dump Database = %s", utils.QuoteIdent(connection.DBName))
	logger.Info("Database Size = %s", connection.GetDBSize())

	masterDumpDir := utils.GetDirForContent(-1)

	globalFilename := fmt.Sprintf("%s/global.sql", masterDumpDir)
	predataFilename := fmt.Sprintf("%s/predata.sql", masterDumpDir)
	postdataFilename := fmt.Sprintf("%s/postdata.sql", masterDumpDir)

	connection.Begin()
	connection.Exec("SET search_path TO pg_catalog")

	tables := GetAllUserTables(connection)
	extTableMap := GetExternalTablesMap(connection)

	logger.Info("Writing global database metadata to %s", globalFilename)
	backupGlobal(globalFilename)
	logger.Info("Global database metadata dump complete")

	logger.Info("Writing pre-data metadata to %s", predataFilename)
	backupPredata(predataFilename, tables, extTableMap)
	logger.Info("Pre-data metadata dump complete")

	logger.Info("Writing data to file")
	backupData(tables, extTableMap)
	logger.Info("Data dump complete")

	logger.Info("Writing post-data metadata to %s", postdataFilename)
	backupPostdata(postdataFilename, tables, extTableMap)
	logger.Info("Post-data metadata dump complete")

	connection.Commit()
}

func backupGlobal(filename string) {
	globalFile := utils.MustOpenFile(filename)

	logger.Verbose("Writing session GUCs to global file")
	gucs := GetSessionGUCs(connection)
	PrintSessionGUCs(globalFile, gucs)

	logger.Verbose("Writing CREATE DATABASE statement to global file")
	dbnames := GetDatabaseNames(connection)
	dbMetadata := GetMetadataForObjectType(connection, "", "datacl", "datdba", "pg_database")
	PrintCreateDatabaseStatement(globalFile, connection.DBName, dbnames, dbMetadata)

	logger.Verbose("Writing database GUCs to global file")
	databaseGucs := GetDatabaseGUCs(connection)
	PrintDatabaseGUCs(globalFile, databaseGucs, connection.DBName)

	logger.Verbose("Writing CREATE RESOURCE QUEUE statements to global file")
	resQueues := GetResourceQueues(connection)
	PrintCreateResourceQueueStatements(globalFile, resQueues)

	logger.Verbose("Writing CREATE ROLE statements to global file")
	roles := GetRoles(connection)
	PrintCreateRoleStatements(globalFile, roles)
}

func backupPredata(filename string, tables []utils.Relation, extTableMap map[string]bool) {
	predataFile := utils.MustOpenFile(filename)
	PrintConnectionString(predataFile, connection.DBName)

	logger.Verbose("Writing session GUCs to predata file")
	gucs := GetSessionGUCs(connection)
	PrintSessionGUCs(predataFile, gucs)

	logger.Verbose("Writing CREATE SCHEMA statements to predata file")
	schemas := GetAllUserSchemas(connection)
	schemaMetadata := GetMetadataForObjectType(connection, "", "nspacl", "nspowner", "pg_namespace")
	PrintCreateSchemaStatements(predataFile, schemas, schemaMetadata)

	types := GetTypeDefinitions(connection)
	typeMetadata := GetMetadataForObjectType(connection, "typnamespace", "", "typowner", "pg_type")
	logger.Verbose("Writing CREATE TYPE statements for shell types to predata file")
	PrintCreateShellTypeStatements(predataFile, types)

	logger.Verbose("Writing CREATE DOMAIN statements to predata file")
	PrintCreateDomainStatements(predataFile, types, typeMetadata)

	funcInfoMap := GetFunctionOidToInfoMap(connection)
	logger.Verbose("Writing CREATE PROCEDURAL LANGUAGE statements to predata file")
	procLangs := GetProceduralLanguages(connection)
	procLangMetadata := GetMetadataForObjectType(connection, "", "lanacl", "lanowner", "pg_language")
	PrintCreateLanguageStatements(predataFile, procLangs, funcInfoMap, procLangMetadata)

	logger.Verbose("Writing CREATE TYPE statements for composite and enum types to predata file")
	PrintCreateCompositeAndEnumTypeStatements(predataFile, types, typeMetadata)

	logger.Verbose("Writing CREATE FUNCTION statements to predata file")
	funcDefs := GetFunctionDefinitions(connection)
	funcMetadata := GetMetadataForObjectType(connection, "pronamespace", "proacl", "proowner", "pg_proc")
	PrintCreateFunctionStatements(predataFile, funcDefs, funcMetadata)

	logger.Verbose("Writing CREATE TYPE statements for base types to predata file")
	PrintCreateBaseTypeStatements(predataFile, types, typeMetadata)

	logger.Verbose("Writing CREATE PROTOCOL statements to predata file")
	protocols := GetExternalProtocols(connection)
	protoMetadata := GetMetadataForObjectType(connection, "", "ptcacl", "ptcowner", "pg_extprotocol")
	PrintCreateExternalProtocolStatements(predataFile, protocols, funcInfoMap, protoMetadata)

	logger.Verbose("Writing CREATE AGGREGATE statements to predata file")
	aggDefs := GetAggregateDefinitions(connection)
	aggMetadata := GetMetadataForObjectType(connection, "", "", "proowner", "pg_proc")
	PrintCreateAggregateStatements(predataFile, aggDefs, funcInfoMap, aggMetadata)

	logger.Verbose("Writing CREATE CAST statements to predata file")
	castDefs := GetCastDefinitions(connection)
	castMetadata := GetCommentsForObjectType(connection, "", "oid", "pg_cast", "pg_cast")
	PrintCreateCastStatements(predataFile, castDefs, castMetadata)

	logger.Verbose("Writing CREATE TABLE statements to predata file")
	relationMetadata := GetMetadataForObjectType(connection, "relnamespace", "relacl", "relowner", "pg_class")
	for _, table := range tables {
		isExternal := extTableMap[table.ToString()]
		tableDef := ConstructDefinitionsForTable(connection, table, isExternal)
		PrintCreateTableStatement(predataFile, table, tableDef, relationMetadata[table.RelationOid])
	}

	logger.Verbose("Writing CREATE VIEW statements to predata file")
	viewDefs := GetViewDefinitions(connection)
	PrintCreateViewStatements(predataFile, viewDefs, relationMetadata)

	logger.Verbose("Writing ADD CONSTRAINT statements to predata file")
	constraints := GetConstraints(connection)
	conMetadata := GetCommentsForObjectType(connection, "", "oid", "pg_constraint", "pg_constraint")
	PrintConstraintStatements(predataFile, constraints, conMetadata)

	logger.Verbose("Writing CREATE SEQUENCE statements to predata file")
	sequenceDefs := GetAllSequences(connection)
	sequenceColumnOwners := GetSequenceColumnOwnerMap(connection)
	PrintCreateSequenceStatements(predataFile, sequenceDefs, sequenceColumnOwners, relationMetadata)
}

func backupData(tables []utils.Relation, extTableMap map[string]bool) {
	for _, table := range tables {
		isExternal := extTableMap[table.ToString()]
		if !isExternal {
			logger.Verbose("Writing data for table %s to file", table.ToString())
			dumpFile := GetTableDumpFilePath(table)
			CopyTableOut(connection, table, dumpFile)
		} else {
			logger.Warn("Skipping data dump of table %s because it is an external table.", table.ToString())
		}
	}
	logger.Verbose("Writing table map file to %s", GetTableMapFilePath())
	WriteTableMapFile(tables)
}

func backupPostdata(filename string, tables []utils.Relation, extTableMap map[string]bool) {
	postdataFile := utils.MustOpenFile(filename)
	PrintConnectionString(postdataFile, connection.DBName)

	logger.Verbose("Writing session GUCs to predata file")
	gucs := GetSessionGUCs(connection)
	PrintSessionGUCs(postdataFile, gucs)

	logger.Verbose("Writing CREATE INDEX statements to postdata file")
	indexNameMap := ConstructImplicitIndexNames(connection)
	indexes := GetIndexDefinitions(connection, indexNameMap)
	indexMetadata := GetCommentsForObjectType(connection, "", "indexrelid", "pg_class", "pg_index")
	PrintCreateIndexStatements(postdataFile, indexes, indexMetadata)

	logger.Verbose("Writing CREATE RULE statements to postdata file")
	rules := GetRuleDefinitions(connection)
	ruleMetadata := GetCommentsForObjectType(connection, "", "oid", "pg_rewrite", "pg_rewrite")
	PrintCreateRuleStatements(postdataFile, rules, ruleMetadata)

	logger.Verbose("Writing CREATE TRIGGER statements to postdata file")
	triggers := GetTriggerDefinitions(connection)
	triggerMetadata := GetCommentsForObjectType(connection, "", "oid", "pg_trigger", "pg_trigger")
	PrintCreateTriggerStatements(postdataFile, triggers, triggerMetadata)
}

func DoTeardown() {
	if r := recover(); r != nil {
		fmt.Println(r)
	}
	if connection != nil {
		connection.Close()
	}
	// TODO: Add logic for error codes based on whether we Abort()ed or not
}

package options

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/greenplum-db/gp-common-go-libs/dbconn"
	"github.com/greenplum-db/gp-common-go-libs/iohelper"
	"github.com/greenplum-db/gpbackup/utils"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type Options struct {
	includedRelations         []string
	isLeafPartitionData       bool
	excludedSchemas           []string
	includedSchemas           []string
	originalIncludedRelations []string
}

func NewOptions(initialFlags *pflag.FlagSet) (*Options, error) {
	includes, err := initialFlags.GetStringArray(utils.INCLUDE_RELATION)
	if err != nil {
		return nil, err
	}

	includedSchemas, err := initialFlags.GetStringSlice(utils.INCLUDE_SCHEMA)
	if err != nil {
		return nil, err
	}

	excludedSchemas, err := initialFlags.GetStringSlice(utils.EXCLUDE_SCHEMA)
	if err != nil {
		return nil, err
	}

	// flag for INCLUDE_RELATION_FILE is mutually exclusive with INCLUDE_RELATION flag
	// so this is not an overwrite, it is a "fresh" setting
	filename, err := initialFlags.GetString(utils.INCLUDE_RELATION_FILE)
	if err != nil {
		return nil, err
	}
	if filename != "" {
		includes, err = setIncludesFromFile(filename, initialFlags)
		if err != nil {
			return nil, err
		}
	}

	err = ValidateCharacters(includes)
	if err != nil {
		return nil, err
	}

	leafPartitionData, err := initialFlags.GetBool(utils.LEAF_PARTITION_DATA)
	if err != nil {
		return nil, err
	}

	return &Options{
		includedRelations:         includes,
		includedSchemas:           includedSchemas,
		excludedSchemas:           excludedSchemas,
		isLeafPartitionData:       leafPartitionData,
		originalIncludedRelations: includes,
	}, nil
}

func setIncludesFromFile(filename string, initialFlags *pflag.FlagSet) ([]string, error) {
	includes, err := iohelper.ReadLinesFromFile(filename)
	if err != nil {
		return nil, err
	}

	// copy any values for flag INCLUDE_RELATION_FILE into global flag for INCLUDE_RELATION
	for _, fqn := range includes {
		err = initialFlags.Set(utils.INCLUDE_RELATION, fqn) //This appends to the slice underlying the flag.
		if err != nil {
			return nil, err
		}
	}
	return includes, nil
}

func (o Options) GetIncludedTables() []string {
	return o.includedRelations
}

func (o Options) GetOriginalIncludedTables() []string {
	return o.originalIncludedRelations
}

func (o Options) IsLeafPartitionData() bool {
	return o.isLeafPartitionData
}

func (o Options) GetIncludedSchemas() []string {
	return o.includedSchemas
}

func (o Options) GetExcludedSchemas() []string {
	return o.excludedSchemas
}

func (o *Options) AddIncludedRelation(relation string) {
	o.includedRelations = append(o.includedRelations, relation)
}

type FqnStruct struct {
	SchemaName string
	TableName  string
}

const QuoteIdent = `SELECT quote_ident('%s') AS schemaname, quote_ident('%s') AS tablename`

func QuoteTableNames(conn *dbconn.DBConn, tableNames []string) ([]string, error) {
	if len(tableNames) == 0 {
		return []string{}, nil
	}
	fqnSlice, err := SeparateSchemaAndTable(tableNames)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)

	for _, fqn := range fqnSlice {
		queryResultTable := make([]FqnStruct, 0)
		query := fmt.Sprintf(QuoteIdent, fqn.SchemaName, fqn.TableName)
		err := conn.Select(&queryResultTable, query)
		if err != nil {
			return nil, err
		}
		quoted := queryResultTable[0].SchemaName + "." + queryResultTable[0].TableName
		result = append(result, quoted)
	}

	return result, nil
}

func SeparateSchemaAndTable(tableNames []string) ([]FqnStruct, error) {
	fqnSlice := make([]FqnStruct, 0)
	for _, fqn := range tableNames {
		parts := strings.Split(fqn, ".")
		if len(parts) > 2 {
			return nil, errors.Errorf("cannot process an Fully Qualified Name with embedded dots yet: %s", fqn)
		}
		if len(parts) < 2 {
			return nil, errors.Errorf("Fully Qualified Names require a minimum of one dot, specifying the schema and table. Cannot process: %s", fqn)
		}
		schema := parts[0]
		table := parts[1]
		if schema == "" || table == "" {
			return nil, errors.Errorf("Fully Qualified Names must specify the schema and table. Cannot process: %s", fqn)
		}

		currFqn := FqnStruct{
			SchemaName: schema,
			TableName:  table,
		}

		fqnSlice = append(fqnSlice, currFqn)
	}

	return fqnSlice, nil
}

func ValidateCharacters(tableList []string) error {
	if len(tableList) == 0 {
		return nil
	}

	validFormat := regexp.MustCompile(`^.+\..+$`)
	for _, fqn := range tableList {
		if !validFormat.Match([]byte(fqn)) {
			return errors.Errorf(`Table %s is not correctly fully-qualified.  Please ensure that it is in the format schema.table, it is quoted appropriately, and it has no preceding or trailing whitespace.`, fqn)
		}
	}

	return nil
}

func (o *Options) ExpandIncludesForPartitions(conn *dbconn.DBConn, flags *pflag.FlagSet) error {
	if len(o.GetIncludedTables()) == 0 {
		return nil
	}

	quotedIncludeRelations, err := QuoteTableNames(conn, o.GetIncludedTables())
	if err != nil {
		return err
	}

	allFqnStructs, err := o.getUserTableRelationsWithIncludeFiltering(conn, quotedIncludeRelations)
	if err != nil {
		return err
	}

	includeSet := map[string]bool{}
	for _, include := range o.GetIncludedTables() {
		includeSet[include] = true
	}

	allFqnSet := map[string]bool{}
	for _, fqnStruct := range allFqnStructs {
		fqn := fmt.Sprintf("%s.%s", fqnStruct.SchemaName, fqnStruct.TableName)
		allFqnSet[fqn] = true
	}

	// set arithmetic: find difference
	diff := make([]string, 0)
	for key := range allFqnSet {
		_, keyExists := includeSet[key]
		if !keyExists {
			diff = append(diff, key)
		}
	}

	for _, fqn := range diff {
		err = flags.Set(utils.INCLUDE_RELATION, fqn)
		if err != nil {
			return err
		}
		o.AddIncludedRelation(fqn)
	}

	return nil
}

func (o Options) getUserTableRelationsWithIncludeFiltering(connectionPool *dbconn.DBConn, includedRelationsQuoted []string) ([]FqnStruct, error) {
	includeOids, err := getOidsFromRelationList(connectionPool, includedRelationsQuoted)
	if err != nil {
		return nil, err
	}

	oidStr := strings.Join(includeOids, ", ")
	childPartitionFilter := ""
	if o.isLeafPartitionData {
		//Get all leaf partition tables whose parents are in the include list
		childPartitionFilter = fmt.Sprintf(`
	OR c.oid IN (
		SELECT
			r.parchildrelid
		FROM pg_partition p
		JOIN pg_partition_rule r ON p.oid = r.paroid
		WHERE p.paristemplate = false
		AND p.parrelid IN (%s))`, oidStr)
	} else {
		//Get only external partition tables whose parents are in the include list
		childPartitionFilter = fmt.Sprintf(`
	OR c.oid IN (
		SELECT
			r.parchildrelid
		FROM pg_partition p
		JOIN pg_partition_rule r ON p.oid = r.paroid
		JOIN pg_exttable e ON r.parchildrelid = e.reloid
		WHERE p.paristemplate = false
		AND e.reloid IS NOT NULL
		AND p.parrelid IN (%s))`, oidStr)
	}

	query := fmt.Sprintf(`
SELECT
	n.nspname AS schemaname,
	c.relname AS tablename
FROM pg_class c
JOIN pg_namespace n
	ON c.relnamespace = n.oid
WHERE %s
AND (
	-- Get tables in the include list
	c.oid IN (%s)
	-- Get parent partition tables whose children are in the include list
	OR c.oid IN (
		SELECT
			p.parrelid
		FROM pg_partition p
		JOIN pg_partition_rule r ON p.oid = r.paroid
		WHERE p.paristemplate = false
		AND r.parchildrelid IN (%s)
	)
	-- Get external partition tables whose siblings are in the include list
	OR c.oid IN (
		SELECT
			r.parchildrelid
		FROM pg_partition_rule r
		JOIN pg_exttable e ON r.parchildrelid = e.reloid
		WHERE r.paroid IN (
			SELECT
				pr.paroid
			FROM pg_partition_rule pr
			WHERE pr.parchildrelid IN (%s)
		)
	)
	%s
)
AND (relkind = 'r')
AND %s
ORDER BY c.oid;`, o.schemaFilterClause("n"), oidStr, oidStr, oidStr, childPartitionFilter, ExtensionFilterClause("c"))

	results := make([]FqnStruct, 0)
	err = connectionPool.Select(&results, query)

	return results, err
}

func getOidsFromRelationList(connectionPool *dbconn.DBConn, quotedRelationNames []string) ([]string, error) {
	relList := utils.SliceToQuotedString(quotedRelationNames)
	query := fmt.Sprintf(`
SELECT
	c.oid AS string
FROM pg_class c
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE quote_ident(n.nspname) || '.' || quote_ident(c.relname) IN (%s)`, relList)

	return dbconn.SelectStringSlice(connectionPool, query)
}

// A list of schemas we don't want to back up, formatted for use in a WHERE clause
func (o Options) schemaFilterClause(namespace string) string {
	schemaFilterClauseStr := ""
	if len(o.GetIncludedSchemas()) > 0 {
		schemaFilterClauseStr = fmt.Sprintf("\nAND %s.nspname IN (%s)", namespace, utils.SliceToQuotedString(o.GetIncludedSchemas()))
	}
	if len(o.GetExcludedSchemas()) > 0 {
		schemaFilterClauseStr = fmt.Sprintf("\nAND %s.nspname NOT IN (%s)", namespace, utils.SliceToQuotedString(o.GetExcludedSchemas()))
	}
	return fmt.Sprintf(`%s.nspname NOT LIKE 'pg_temp_%%' AND %s.nspname NOT LIKE 'pg_toast%%' AND %s.nspname NOT IN ('gp_toolkit', 'information_schema', 'pg_aoseg', 'pg_bitmapindex', 'pg_catalog') %s`, namespace, namespace, namespace, schemaFilterClauseStr)
}

func ExtensionFilterClause(namespace string) string {
	oidStr := "oid"
	if namespace != "" {
		oidStr = fmt.Sprintf("%s.oid", namespace)
	}

	return fmt.Sprintf("%s NOT IN (select objid from pg_depend where deptype = 'e')", oidStr)
}

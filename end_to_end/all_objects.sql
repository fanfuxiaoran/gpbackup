--
-- Greenplum Database database dump
--

SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

SET default_with_oids = false;

--
--

CREATE SCHEMA schema2;



--
--

CREATE PROCEDURAL LANGUAGE plperl;


SET search_path = public, pg_catalog;

--
--

CREATE TYPE base_type;


--
--

CREATE FUNCTION base_fn_in(cstring) RETURNS base_type
    AS $$boolin$$
    LANGUAGE internal NO SQL;



--
--

CREATE FUNCTION base_fn_out(base_type) RETURNS cstring
    AS $$boolout$$
    LANGUAGE internal NO SQL;



--
--

CREATE TYPE base_type (
    INTERNALLENGTH = variable,
    INPUT = base_fn_in,
    OUTPUT = base_fn_out,
    ALIGNMENT = int4,
    STORAGE = plain
);



--
--

CREATE TYPE composite_type AS (
	name integer,
	name1 integer,
	name2 text
);



--
--

CREATE TYPE enum_type AS ENUM (
    '750582',
    '750583',
    '750584'
);



--
--

CREATE FUNCTION casttoint(text) RETURNS integer
    AS $_$
SELECT cast($1 as integer);
$_$
    LANGUAGE sql IMMUTABLE STRICT CONTAINS SQL;



--
--

CREATE FUNCTION dup(integer DEFAULT 42, OUT f1 integer, OUT f2 text) RETURNS record
    AS $_$
SELECT $1, CAST($1 AS text) || ' is text'
$_$
    LANGUAGE sql CONTAINS SQL;



--
--

CREATE FUNCTION mypre_accum(numeric, numeric) RETURNS numeric
    AS $_$
select $1 + $2
$_$
    LANGUAGE sql IMMUTABLE STRICT CONTAINS SQL;



--
--

CREATE FUNCTION mysfunc_accum(numeric, numeric, numeric) RETURNS numeric
    AS $_$
select $1 + $2 + $3
$_$
    LANGUAGE sql IMMUTABLE STRICT CONTAINS SQL;



--
--

CREATE FUNCTION plusone(x text) RETURNS text
    AS $$
BEGIN
    RETURN x || 'x';
END;
$$
    LANGUAGE plpgsql NO SQL;



--
--

CREATE FUNCTION plusone(x character varying) RETURNS character varying
    AS $$
BEGIN
    RETURN x || 'a';
END;
$$
    LANGUAGE plpgsql NO SQL
    SET standard_conforming_strings TO 'on'
    SET client_min_messages TO 'notice'
    SET search_path TO public;



--
--

CREATE FUNCTION return_enum_as_array(anyenum, anyelement, anyelement) RETURNS TABLE(ae anyenum, aa anyarray)
    AS $_$
SELECT $1, array[$2, $3]
$_$
    LANGUAGE sql STABLE CONTAINS SQL;



--
--

CREATE AGGREGATE agg_prefunc(numeric, numeric) (
    SFUNC = mysfunc_accum,
    STYPE = numeric,
    INITCOND = '0',
    PREFUNC = mypre_accum
);



--
--

CREATE AGGREGATE agg_test(integer) (
    SFUNC = int4xor,
    STYPE = integer,
    INITCOND = '0'
);



--
--

CREATE OPERATOR #### (
    PROCEDURE = numeric_fac,
    LEFTARG = bigint
);



--
--

CREATE OPERATOR CLASS test_op_class
    FOR TYPE uuid USING hash AS
    STORAGE uuid;



--
--

CREATE OPERATOR FAMILY testfam USING gist;



--
--

CREATE OPERATOR CLASS testclass
    FOR TYPE uuid USING gist FAMILY testfam AS
    OPERATOR 1 =(uuid,uuid) RECHECK ,
    OPERATOR 2 <(uuid,uuid) ,
    FUNCTION 1 abs(integer) ,
    FUNCTION 2 int4out(integer);



SET default_tablespace = '';

--
--

CREATE TABLE bar (
    i integer NOT NULL,
    j text,
    k smallint NOT NULL,
    l character varying(20)
) DISTRIBUTED BY (i);



--
--

COPY bar (i, j, k, l) FROM stdin;
\.


--
--

CREATE TABLE foo (
    k text,
    i integer,
    j text
) DISTRIBUTED RANDOMLY;



--
--

COPY foo (k, i, j) FROM stdin;
\.


--
--

CREATE TABLE foo2 (
    k text,
    l character varying(20)
)
INHERITS (foo) DISTRIBUTED RANDOMLY;



--
--

COPY foo2 (k, i, j, l) FROM stdin;
\.


SET search_path = schema2, pg_catalog;

--
--

CREATE TABLE foo3 (
    m double precision
)
INHERITS (public.foo2) DISTRIBUTED RANDOMLY;



SET search_path = public, pg_catalog;

--
--

CREATE TABLE foo4 (
    n integer
)
INHERITS (schema2.foo3) DISTRIBUTED RANDOMLY;



--
--

COPY foo4 (k, i, j, l, m, n) FROM stdin;
\.


--
--

CREATE TABLE gpcrondump_history (
    rec_date timestamp without time zone,
    start_time character(8),
    end_time character(8),
    options text,
    dump_key character varying(20),
    dump_exit_status smallint,
    script_exit_status smallint,
    exit_text character varying(10)
) DISTRIBUTED BY (rec_date);



--
--

COPY gpcrondump_history (rec_date, start_time, end_time, options, dump_key, dump_exit_status, script_exit_status, exit_text) FROM stdin;
\.


SET search_path = schema2, pg_catalog;

--
--

COPY foo3 (k, i, j, l, m) FROM stdin;
\.


--
--

CREATE TABLE noatts (
) DISTRIBUTED RANDOMLY;



--
--

COPY noatts  FROM stdin;
\.


SET search_path = public, pg_catalog;

--
--

CREATE TABLE pk_table (
    a integer NOT NULL
) DISTRIBUTED BY (a);



--
--

COPY pk_table (a) FROM stdin;
\.


--
--

CREATE TABLE reference_table (
    a integer,
    b integer
) DISTRIBUTED BY (a);



--
--

COPY reference_table (a, b) FROM stdin;
\.


SET search_path = schema2, pg_catalog;

--
--

CREATE TABLE prime (
    i integer NOT NULL,
    j integer
) DISTRIBUTED BY (i);



--
--

COPY prime (i, j) FROM stdin;
\.


SET search_path = public, pg_catalog;

--
--

CREATE TABLE rule_table1 (
    i integer
) DISTRIBUTED BY (i);



--
--

COPY rule_table1 (i) FROM stdin;
\.


SET search_path = pg_catalog;

--
-- Name: CAST (text AS integer); Type: CAST; Schema: pg_catalog; Owner: 
--

CREATE CAST (text AS integer) WITH FUNCTION public.casttoint(text) AS ASSIGNMENT;


--
-- Name: CAST (text AS integer); Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON CAST (text AS integer) IS 'sample cast';


SET search_path = public, pg_catalog;

--
--

CREATE TABLE trigger_table1 (
    i integer
) DISTRIBUTED BY (i);



--
--

COPY trigger_table1 (i) FROM stdin;
\.


--
--

CREATE TABLE uniq (
    i integer
) DISTRIBUTED BY (i);



--
--

COPY uniq (i) FROM stdin;
\.


SET search_path = schema2, pg_catalog;

--
--

CREATE TABLE with_multiple_check (
    a integer,
    b character varying(40),
    CONSTRAINT con1 CHECK (((a > 99) AND ((b)::text <> ''::text)))
) DISTRIBUTED BY (a);



--
--

COPY with_multiple_check (a, b) FROM stdin;
\.


SET search_path = public, pg_catalog;

--
--

CREATE CONVERSION testconv FOR 'LATIN1' TO 'MULE_INTERNAL' FROM latin1_to_mic;



--
--

CREATE TEXT SEARCH DICTIONARY testdictionary (
    TEMPLATE = pg_catalog.snowball,
    language = 'russian', stopwords = 'russian' );



--
--

CREATE TEXT SEARCH CONFIGURATION testconfiguration (
    PARSER = pg_catalog."default" );



--
-- Name: testtemplate; Type: TEXT SEARCH TEMPLATE; Schema: public; Owner: 
--

CREATE TEXT SEARCH TEMPLATE testtemplate (
    LEXIZE = dsimple_lexize );


--
--

CREATE VIEW test_view AS
    SELECT pk_table.a FROM pk_table;



--
--

CREATE VIEW view_view AS
    SELECT test_view.a FROM test_view;



SET search_path = schema2, pg_catalog;

--
--

CREATE SEQUENCE seq_one
    START WITH 3
    INCREMENT BY 1
    NO MAXVALUE
    NO MINVALUE
    CACHE 1;



--
--

ALTER SEQUENCE seq_one OWNED BY prime.j;


--
--

SELECT pg_catalog.setval('seq_one', 3, false);


SET search_path = public, pg_catalog;

--
-- Name: testparser; Type: TEXT SEARCH PARSER; Schema: public; Owner: 
--

CREATE TEXT SEARCH PARSER testparser (
    START = prsd_start,
    GETTOKEN = prsd_nexttoken,
    END = prsd_end,
    LEXTYPES = prsd_lextype );


--
--

ALTER TABLE ONLY pk_table
    ADD CONSTRAINT pk_table_pkey PRIMARY KEY (a);


--
--

ALTER TABLE ONLY uniq
    ADD CONSTRAINT uniq_i_key UNIQUE (i);


SET search_path = schema2, pg_catalog;

--
--

ALTER TABLE ONLY prime
    ADD CONSTRAINT prime_pkey PRIMARY KEY (i);


SET search_path = public, pg_catalog;

--
--

CREATE INDEX simple_table_idx1 ON foo4 USING btree (n);


--
--

CREATE RULE double_insert AS ON INSERT TO rule_table1 DO INSERT INTO rule_table1 VALUES (1);


--
--

CREATE TRIGGER sync_trigger_table1
    AFTER INSERT OR DELETE OR UPDATE ON trigger_table1
    FOR EACH STATEMENT
    EXECUTE PROCEDURE flatfile_update_trigger();


--
--

ALTER TABLE ONLY reference_table
    ADD CONSTRAINT reference_table_b_fkey FOREIGN KEY (b) REFERENCES pk_table(a);


--
--

REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO PUBLIC;


--
-- Greenplum Database database dump complete
--


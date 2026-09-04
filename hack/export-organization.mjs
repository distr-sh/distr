#!/usr/bin/env node

import {spawnSync} from 'node:child_process';
import {closeSync, openSync, renameSync, unlinkSync, writeSync} from 'node:fs';

// Tables that exist independently of any organization and are identified across instances by a
// natural key instead of their id. References to them are exported as a lookup on that key, so an
// import links to the account or agent version the target instance already has.
const SHARED_TABLES = new Map([
  ['useraccount', 'email'],
  ['agentversion', 'name'],
]);

// Instance-scoped values that must not be carried over to another instance.
const COLUMN_OVERRIDES = new Map([
  // Granting instance-wide staff access through an organization import would be privilege escalation.
  ['useraccount.is_super_admin', () => `'false'`],
  [
    'useraccount.last_used_organization_id',
    (orgLiteral) =>
      `quote_nullable(CASE WHEN t.last_used_organization_id = ${orgLiteral}::uuid` +
      ` THEN t.last_used_organization_id END)`,
  ],
]);

// Logs, telemetry and transient state. Excluded by default because they hold by far the most rows
// while nothing in the product depends on their history. The two log record tables predate the move
// of log storage to Loki and only hold what an instance recorded before it; the telemetry tables are
// repopulated by the agents once they reconnect.
const DEFAULT_EXCLUDED = new Map([
  ['artifactversionpull', 'artifact pull audit log'],
  ['deploymentlogrecord', 'deployment logs recorded before Loki'],
  ['deploymentmetrics', 'deployment resource telemetry'],
  ['deploymentresourcemetrics', 'deployment resource telemetry'],
  ['deploymentrevisionstatus', 'deployment status history'],
  ['deploymenttargetdiskmetrics', 'agent disk telemetry'],
  ['deploymenttargetlogrecord', 'agent logs recorded before Loki'],
  ['deploymenttargetmetrics', 'agent telemetry'],
  ['notificationrecord', 'sent alert history'],
  ['oidcstate', 'transient OIDC login state'],
]);

const USAGE = `Usage: hack/export-organization.mjs <organization-id> [options]

Exports every row belonging to one organization as a plain .sql file that can be imported into
another Distr instance with psql. The set of exported tables is derived from the database itself:
it is the closure of ON DELETE CASCADE foreign keys starting at the organization row, which is the
same set of rows that deleting the organization would remove.

Options:
  --database-url=<url>  Postgres connection string (default: $DATABASE_URL)
  --psql=<command>      Command to reach psql, instead of a local psql and a connection string.
                        Avoids a port forward against a cluster, which is not reliable enough for
                        a long transfer, e.g.
                          --psql="kubectl exec -i -n distr postgres-0 -- psql -U postgres -d distr"
  --output=<file>       Output file (default: distr-org-<organization-id>.sql, - for stdout)
  --all                 Also export the telemetry and transient tables excluded by default
  --exclude=<t1,t2>     Additionally exclude these tables
  --no-count            Skip the pre-flight row count, which costs about as much as the export
                        itself and only serves the summary. Worth it over a slow connection.
  --help

Excluded by default:
${[...DEFAULT_EXCLUDED].map(([table, reason]) => `  ${table.padEnd(28)} ${reason}`).join('\n')}

Not covered by the export, since it does not live in Postgres:
  - OCI registry blobs (S3 bucket), which hack/copy-artifact-blobs.sh copies
  - deployment and deployment target logs (Loki)
`;

const FIELD_SEP = '\x1f';

function fail(message) {
  console.error(`error: ${message}`);
  process.exit(1);
}

function parseArgs(argv) {
  const options = {
    orgId: null,
    databaseUrl: process.env.DATABASE_URL,
    psql: null,
    output: null,
    all: false,
    exclude: [],
    count: true,
  };
  for (const arg of argv) {
    if (arg === '--help' || arg === '-h') {
      process.stdout.write(USAGE);
      process.exit(0);
    } else if (arg === '--all') {
      options.all = true;
    } else if (arg === '--no-count') {
      options.count = false;
    } else if (arg.startsWith('--database-url=')) {
      options.databaseUrl = arg.slice('--database-url='.length);
    } else if (arg.startsWith('--psql=')) {
      options.psql = arg.slice('--psql='.length).split(/\s+/).filter(Boolean);
    } else if (arg.startsWith('--output=')) {
      options.output = arg.slice('--output='.length);
    } else if (arg.startsWith('--exclude=')) {
      options.exclude.push(...arg.slice('--exclude='.length).split(',').filter(Boolean));
    } else if (arg.startsWith('-')) {
      fail(`unknown option: ${arg}\n\n${USAGE}`);
    } else if (options.orgId === null) {
      options.orgId = arg;
    } else {
      fail(`unexpected argument: ${arg}`);
    }
  }
  if (!options.orgId) fail(`missing organization id\n\n${USAGE}`);
  if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(options.orgId)) {
    fail(`not a uuid: ${options.orgId}`);
  }
  if (!options.psql) {
    if (!options.databaseUrl) fail('no database url: pass --database-url, --psql or set DATABASE_URL');
    options.psql = ['psql', options.databaseUrl];
  }
  options.output ??= `distr-org-${options.orgId}.sql`;
  return options;
}

function ident(name) {
  return `"${name.replace(/"/g, '""')}"`;
}

function literal(value) {
  return `'${String(value).replace(/'/g, "''")}'`;
}

function query(psql, sql) {
  const [command, ...args] = psql;
  const result = spawnSync(
    command,
    [...args, '-X', '-q', '-A', '-t', '-F', FIELD_SEP, '-v', 'ON_ERROR_STOP=1', '-c', sql],
    {
      encoding: 'utf-8',
      maxBuffer: 1024 * 1024 * 1024,
    }
  );
  if (result.error) fail(`could not run ${command}: ${result.error.message}`);
  if (result.status !== 0) fail(`${command} failed:\n${result.stderr.trim()}`);
  return result.stdout
    .split('\n')
    .filter((line) => line !== '')
    .map((line) => line.split(FIELD_SEP));
}

function readSchema(psql) {
  const columns = new Map();
  for (const [table, column] of query(
    psql,
    `SELECT c.relname, a.attname
     FROM pg_class c
     JOIN pg_namespace n ON n.oid = c.relnamespace
     JOIN pg_attribute a ON a.attrelid = c.oid
     WHERE n.nspname = 'public' AND c.relkind = 'r'
       AND a.attnum > 0 AND NOT a.attisdropped AND a.attgenerated = ''
     ORDER BY c.relname, a.attnum`
  )) {
    if (!columns.has(table)) columns.set(table, []);
    columns.get(table).push(column);
  }

  const primaryKeys = new Map(
    query(
      psql,
      `SELECT c.relname, (
         SELECT string_agg(a.attname, ',' ORDER BY k.ord)
         FROM unnest(co.conkey) WITH ORDINALITY k(att, ord)
         JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = k.att
       )
       FROM pg_constraint co
       JOIN pg_class c ON c.oid = co.conrelid
       JOIN pg_namespace n ON n.oid = c.relnamespace
       WHERE co.contype = 'p' AND n.nspname = 'public'`
    ).map(([table, cols]) => [table, cols.split(',')])
  );

  const foreignKeys = query(
    psql,
    `SELECT ch.relname, p.relname, co.confdeltype::text, (
       SELECT string_agg(a.attname, ',' ORDER BY k.ord)
       FROM unnest(co.conkey) WITH ORDINALITY k(att, ord)
       JOIN pg_attribute a ON a.attrelid = ch.oid AND a.attnum = k.att
     ), (
       SELECT string_agg(a.attname, ',' ORDER BY k.ord)
       FROM unnest(co.confkey) WITH ORDINALITY k(att, ord)
       JOIN pg_attribute a ON a.attrelid = p.oid AND a.attnum = k.att
     ), (
       SELECT bool_or(a.attnotnull)
       FROM unnest(co.conkey) k(att)
       JOIN pg_attribute a ON a.attrelid = ch.oid AND a.attnum = k.att
     )
     FROM pg_constraint co
     JOIN pg_class ch ON ch.oid = co.conrelid
     JOIN pg_class p ON p.oid = co.confrelid
     JOIN pg_namespace n ON n.oid = ch.relnamespace
     WHERE co.contype = 'f' AND n.nspname = 'public'
     ORDER BY ch.relname, co.conname`
  ).map(([child, parent, onDelete, childCols, parentCols, notNull]) => ({
    child,
    parent,
    cascades: onDelete === 'c',
    columns: childCols.split(','),
    parentColumns: parentCols.split(','),
    notNull: notNull === 't',
  }));

  return {columns, primaryKeys, foreignKeys};
}

// The rows that deleting the organization would delete: every table that reaches the organization
// through ON DELETE CASCADE foreign keys. Deliberately ignores the other delete actions, which mark
// a plain reference rather than ownership (useraccount.last_used_organization_id, for example).
function ownedTables(foreignKeys) {
  const owned = new Set(['organization']);
  for (let grew = true; grew;) {
    grew = false;
    for (const fk of foreignKeys) {
      if (fk.cascades && owned.has(fk.parent) && !owned.has(fk.child)) {
        owned.add(fk.child);
        grew = true;
      }
    }
  }
  return owned;
}

function topologicalOrder(tables, foreignKeys) {
  const dependencies = new Map([...tables].map((table) => [table, new Set()]));
  for (const fk of foreignKeys) {
    if (fk.child !== fk.parent && tables.has(fk.child) && tables.has(fk.parent)) {
      dependencies.get(fk.child).add(fk.parent);
    }
  }

  const order = [];
  const emitted = new Set();
  while (order.length < dependencies.size) {
    const ready = [...dependencies]
      .filter(([table, parents]) => !emitted.has(table) && [...parents].every((p) => emitted.has(p)))
      .map(([table]) => table)
      .sort();
    if (ready.length === 0) {
      const remaining = [...dependencies.keys()].filter((table) => !emitted.has(table));
      fail(`foreign keys form a cycle between: ${remaining.join(', ')}`);
    }
    for (const table of ready) {
      order.push(table);
      emitted.add(table);
    }
  }
  return order;
}

class Generator {
  constructor(schema, orgId, exported) {
    this.schema = schema;
    this.orgLiteral = literal(orgId);
    this.exported = exported;
  }

  cascadeParents(table) {
    return this.schema.foreignKeys.filter((fk) => fk.child === table && fk.cascades && fk.child !== fk.parent);
  }

  // A boolean expression selecting the organization's rows of `table`, with all columns qualified by
  // an alias so that a nested level can never silently correlate with an enclosing one.
  rowsOf(table, alias, depth = 1) {
    if (table === 'organization') return `${alias}.id = ${this.orgLiteral}`;

    const parents = this.cascadeParents(table).filter(
      (fk) => this.exported.has(fk.parent) && !SHARED_TABLES.has(fk.parent)
    );
    const direct = parents.find((fk) => fk.parent === 'organization');
    if (direct) return `${alias}.${ident(direct.columns[0])} = ${this.orgLiteral}`;
    if (parents.length === 0) fail(`cannot determine the organization's rows of ${table}`);

    return parents
      .map((fk) => {
        const parentAlias = `p${depth}`;
        const columns = fk.columns.map((c) => `${alias}.${ident(c)}`).join(', ');
        const parentColumns = fk.parentColumns.map((c) => `${parentAlias}.${ident(c)}`).join(', ');
        const parentRows = this.rowsOf(fk.parent, parentAlias, depth + 1);
        return `(${columns}) IN (SELECT ${parentColumns} FROM ${ident(fk.parent)} ${parentAlias} WHERE ${parentRows})`;
      })
      .join(' OR ');
  }

  // Rows of a shared table are exported because an exported row points at them.
  rowsOfShared(table, alias) {
    const references = this.schema.foreignKeys.filter(
      (fk) => fk.parent === table && this.exported.has(fk.child) && !SHARED_TABLES.has(fk.child)
    );
    if (references.length === 0) return 'false';
    const referenced = references[0].parentColumns;
    if (references.some((fk) => String(fk.parentColumns) !== String(referenced))) {
      fail(`${table} is referenced by more than one key; a shared table must be referenced by only one`);
    }
    const columns = referenced.map((c) => `${alias}.${ident(c)}`).join(', ');
    const sources = references
      .map(
        (fk) =>
          `SELECT ${fk.columns.map((c) => `s.${ident(c)}`).join(', ')}` +
          ` FROM ${ident(fk.child)} s WHERE ${this.rowsOf(fk.child, 's', 1)}`
      )
      .join('\n      UNION ');
    return `(${columns}) IN (\n      ${sources}\n    )`;
  }

  // Whether the ownership predicate covers every row of `table` that an exported row may reference.
  // It does not when ownership is optional: a `file` row holding a user's avatar or a customer
  // organization's logo has no organization_id and is therefore unreachable from the organization.
  isComplete(table, seen = new Set()) {
    if (table === 'organization' || SHARED_TABLES.has(table)) return true;
    if (seen.has(table)) return false;
    seen.add(table);

    const parents = this.cascadeParents(table).filter(
      (fk) => this.exported.has(fk.parent) && !SHARED_TABLES.has(fk.parent)
    );
    const direct = parents.find((fk) => fk.parent === 'organization');
    if (direct) return direct.notNull;
    return parents.some((fk) => fk.notNull && this.isComplete(fk.parent, seen));
  }

  // Rows of an optionally owned table that are only reachable through a reference from an exported
  // row, so that no exported row ends up pointing at something the import does not contain.
  referencedRowsOf(table) {
    if (SHARED_TABLES.has(table) || this.isComplete(table)) return [];
    return this.schema.foreignKeys
      .filter((fk) => fk.parent === table && fk.child !== table && this.exported.has(fk.child) && !fk.cascades)
      .map((fk) => {
        const columns = fk.parentColumns.map((c) => `t.${ident(c)}`).join(', ');
        const sourceColumns = fk.columns.map((c) => `s.${ident(c)}`).join(', ');
        const sourceRows = SHARED_TABLES.has(fk.child)
          ? this.rowsOfShared(fk.child, 's')
          : this.rowsOf(fk.child, 's', 1);
        return `(${columns}) IN (SELECT ${sourceColumns} FROM ${ident(fk.child)} s WHERE ${sourceRows})`;
      });
  }

  where(table) {
    const rows = SHARED_TABLES.has(table) ? this.rowsOfShared(table, 't') : this.rowsOf(table, 't');
    return [rows, ...this.referencedRowsOf(table)].map((clause) => `(${clause})`).join('\n    OR ');
  }

  columnExpression(table, column) {
    const override = COLUMN_OVERRIDES.get(`${table}.${column}`);
    if (override) return override(this.orgLiteral);

    const reference = this.schema.foreignKeys.find(
      (fk) => fk.child === table && SHARED_TABLES.has(fk.parent) && fk.columns.length === 1 && fk.columns[0] === column
    );
    if (reference) {
      const key = ident(SHARED_TABLES.get(reference.parent));
      const target = ident(reference.parent);
      const lookup =
        `'(SELECT ${ident(reference.parentColumns[0])} FROM ${target} WHERE ${key} = '` +
        ` || quote_literal(r.${key}) || ')'`;
      return (
        `coalesce((SELECT ${lookup} FROM ${target} r` +
        ` WHERE r.${ident(reference.parentColumns[0])} = t.${ident(column)}), 'NULL')`
      );
    }

    const dangling = this.schema.foreignKeys.find(
      (fk) =>
        fk.child === table &&
        !this.exported.has(fk.parent) &&
        !SHARED_TABLES.has(fk.parent) &&
        fk.columns.includes(column)
    );
    if (dangling) {
      if (dangling.notNull) {
        fail(
          `${table}.${column} requires ${dangling.parent}, which is excluded from the export.` +
            ` Either keep ${dangling.parent} or also exclude ${table}.`
        );
      }
      return `'NULL'`;
    }

    return `quote_nullable(t.${ident(column)})`;
  }

  // Emits one INSERT per row. Every value is produced by quote_nullable, whose output is a literal of
  // the column's text representation, which Postgres coerces back to the column type on import. That
  // keeps bytea, arrays, json and enums correct without the script knowing any type.
  statement(table) {
    const columns = this.schema.columns.get(table);
    const conflict = SHARED_TABLES.has(table) ? ' ON CONFLICT DO NOTHING' : '';
    const values = columns
      .map((column, i) => `${i === 0 ? '' : `', ' || `}${this.columnExpression(table, column)}`)
      .join(' || ');
    const insert =
      `'INSERT INTO ${ident(table)} (${columns.map(ident).join(', ')}) VALUES (' || ` + `${values} || ')${conflict};'`;
    const abort = `'DO $abort$ BEGIN RAISE EXCEPTION ''export bug: null row literal in ${table}''; END $abort$;'`;
    const order = (this.schema.primaryKeys.get(table) ?? columns).map((c) => `t.${ident(c)}`).join(', ');

    return (
      `SELECT E'\\n-- ${table}';\n` +
      `SELECT coalesce(${insert}, ${abort})\n` +
      `FROM ${ident(table)} t\n` +
      `WHERE ${this.where(table)}\nORDER BY ${order};\n`
    );
  }

  countQuery(tables) {
    return (
      tables
        .map(
          (table) =>
            `SELECT ${literal(table)} AS name, count(*) AS rows FROM ${ident(table)} t WHERE ${this.where(table)}`
        )
        .join('\nUNION ALL ') + ';'
    );
  }
}

const options = parseArgs(process.argv.slice(2));
const schema = readSchema(options.psql);

const [migration] = query(options.psql, 'SELECT version, dirty FROM schema_migrations');
if (!migration) fail('no schema_migrations row: is this a Distr database?');
if (migration[1] === 't') fail(`database is at a dirty migration (version ${migration[0]})`);

const [organization] = query(
  options.psql,
  `SELECT name, created_at FROM organization WHERE id = ${literal(options.orgId)}`
);
if (!organization) fail(`no organization with id ${options.orgId}`);

const excluded = new Set([
  ...(options.all ? [] : [...DEFAULT_EXCLUDED.keys()].filter((table) => schema.columns.has(table))),
  ...options.exclude,
  'schema_migrations',
]);
for (const table of options.exclude) {
  if (!schema.columns.has(table)) fail(`--exclude names an unknown table: ${table}`);
}

const owned = ownedTables(schema.foreignKeys);
const exported = new Set([...owned, ...SHARED_TABLES.keys()].filter((table) => !excluded.has(table)));

// A table whose every owning parent is excluded has no identifiable set of rows left, so excluding
// one table excludes everything owned through it.
const dependent = [];
for (let shrank = true; shrank;) {
  shrank = false;
  for (const table of exported) {
    if (table === 'organization' || SHARED_TABLES.has(table)) continue;
    const owners = schema.foreignKeys.filter(
      (fk) =>
        fk.child === table &&
        fk.cascades &&
        fk.child !== fk.parent &&
        exported.has(fk.parent) &&
        !SHARED_TABLES.has(fk.parent)
    );
    if (owners.length === 0) {
      exported.delete(table);
      dependent.push(table);
      shrank = true;
    }
  }
}
for (const table of dependent) excluded.add(table);

const order = topologicalOrder(exported, schema.foreignKeys);
const generator = new Generator(schema, options.orgId, exported);

const skipped = [...schema.columns.keys()].filter((table) => !exported.has(table) && !excluded.has(table)).sort();

console.error(`Organization: ${organization[0]} (${options.orgId})`);
console.error(`Tables:       ${order.length} exported, ${excluded.size - 1} excluded, ${skipped.length} not owned`);
if (dependent.length > 0) {
  console.error(`Also excluded, since they are owned through an excluded table: ${dependent.sort().join(', ')}`);
}
console.error(`Not owned by any organization, so not exported: ${skipped.join(', ')}`);

let total = null;
if (options.count) {
  const counts = query(options.psql, generator.countQuery(order));
  total = counts.reduce((sum, [, n]) => sum + Number(n), 0);
  for (const [table, n] of counts.filter(([, n]) => Number(n) > 0).sort((a, b) => Number(b[1]) - Number(a[1]))) {
    console.error(`  ${table.padEnd(44)} ${String(n).padStart(9)}`);
  }
  console.error(`  ${'total'.padEnd(44)} ${String(total).padStart(9)}`);
}

const generated = order.map((table) => generator.statement(table)).join('\n');

const header = `--
-- Distr organization export
--
-- organization: ${organization[0]} (${options.orgId}), created ${organization[1]}
-- exported at:  ${new Date().toISOString()}
-- schema:       migration version ${migration[0]}
${total === null ? '' : `-- rows:         ${total}\n`}--
-- Import into another Distr instance with:
--
--     psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f ${options.output === '-' ? 'this-file.sql' : options.output}
--
-- The import runs in a single transaction and aborts unless the target instance is at the same
-- migration version. It is not idempotent: importing twice fails on the organization's primary key.
--
-- A complete file ends with the line "-- end of export". If it does not, the transfer was cut off:
-- the import would then reach the end of the file with the transaction still open, which rolls
-- everything back and, because psql reports no error for that, still exits successfully.
--
-- Accounts and agent versions are shared between organizations, so they are inserted with
-- ON CONFLICT DO NOTHING and referenced by email respectively name. An account that already exists
-- on the target instance keeps its current password, and the import adds it to this organization.
--
-- Not included, because it does not live in Postgres:
--   - OCI registry blobs (S3 bucket), copy them with hack/copy-artifact-blobs.sh ${options.orgId}
--   - deployment and deployment target logs (Loki)
--
-- Not included, because it belongs to no organization:
${skipped.map((table) => `--   ${table}`).join('\n')}
${
  excluded.size > 1
    ? `--\n-- Excluded tables:\n${[...excluded]
        .filter((table) => table !== 'schema_migrations')
        .sort()
        .map((table) => `--   ${table}${DEFAULT_EXCLUDED.has(table) ? ` (${DEFAULT_EXCLUDED.get(table)})` : ''}`)
        .join('\n')}\n`
    : ''
}--
-- This file contains password hashes, MFA secrets, access tokens, license signing material and
-- SMTP and OIDC credentials. Treat it as a secret.
--

\\set ON_ERROR_STOP on

BEGIN;

DO $guard$
DECLARE target bigint;
BEGIN
  SELECT version INTO target FROM schema_migrations;
  IF target IS DISTINCT FROM ${migration[0]} THEN
    RAISE EXCEPTION 'schema mismatch: export is at migration version ${migration[0]}, target is at %', target;
  END IF;
END $guard$;
`;

const footer = `
COMMIT;

-- end of export
`;

// Written to a temporary name and renamed only once psql has finished, so that a transfer which
// dies halfway through - a port forward against a cluster regularly does - cannot leave a truncated
// file behind under the name the operator is going to import.
const streamToStdout = options.output === '-';
const partial = `${options.output}.part`;
const output = streamToStdout ? process.stdout.fd : openSync(partial, 'w', 0o600);
writeSync(output, header);

// The generating query goes in through stdin rather than a file, since with --psql the psql that
// has to read it runs on another machine, where no path of ours exists.
const [command, ...args] = options.psql;
const run = spawnSync(command, [...args, '-X', '-q', '-A', '-t', '-v', 'ON_ERROR_STOP=1'], {
  input: generated,
  stdio: ['pipe', output, 'inherit'],
});

if (run.error || run.status !== 0) {
  if (!streamToStdout) {
    closeSync(output);
    unlinkSync(partial);
  }
  fail(run.error ? `could not run ${command}: ${run.error.message}` : `${command} failed while generating the export`);
}

writeSync(output, footer);
if (!streamToStdout) {
  closeSync(output);
  renameSync(partial, options.output);
  console.error(`\nWrote ${options.output}`);
}

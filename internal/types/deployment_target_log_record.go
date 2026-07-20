package types

// DeploymentTargetLogRecord was removed: deployment target log records are now stored in and served
// from the log store (Loki) instead of the DeploymentTargetLogRecord Postgres table. The backing
// table is kept around temporarily so we can revert to the previous version if needed and provide
// manual exports for customers on request.
//
// TODO: Drop the DeploymentTargetLogRecord table once we are confident we no longer need to revert
// or export the historical records.

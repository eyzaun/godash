-- GoDash Database Initialization Script (Simplified)
-- This script sets up the initial database structure and configurations

-- Create extensions if available
DO $ 
BEGIN
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
EXCEPTION WHEN OTHERS THEN
    NULL; -- Extension might not be available, continue
END $;

-- Set timezone
SET timezone = 'UTC';

-- Create custom types for better data integrity (simplified)
DO $ 
BEGIN
    CREATE TYPE alert_severity AS ENUM ('low', 'medium', 'high', 'critical');
EXCEPTION WHEN duplicate_object THEN
    NULL; -- Type already exists
END $;

DO $ 
BEGIN
    CREATE TYPE alert_condition AS ENUM ('>', '<', '>=', '<=', '==', '!=');
EXCEPTION WHEN duplicate_object THEN
    NULL; -- Type already exists
END $;

-- Grant permissions to godash user
GRANT ALL PRIVILEGES ON DATABASE godash TO godash;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO godash;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO godash;

-- Grant permissions for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO godash;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO godash;

-- Create a function to get database statistics
CREATE OR REPLACE FUNCTION get_database_stats()
RETURNS TABLE(
    table_name TEXT,
    row_count BIGINT,
    size_bytes BIGINT
) AS $
BEGIN
    RETURN QUERY
    SELECT 
        schemaname || '.' || tablename AS table_name,
        COALESCE(n_tup_ins - n_tup_del, 0) AS row_count,
        pg_total_relation_size(schemaname||'.'||tablename) AS size_bytes
    FROM pg_stat_user_tables
    ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
END;
$ LANGUAGE plpgsql;

-- Create a view for system health monitoring
CREATE OR REPLACE VIEW system_health_summary AS
SELECT 
    'database' AS component,
    'healthy' AS status,
    json_build_object(
        'connections', (SELECT count(*) FROM pg_stat_activity WHERE datname = current_database()),
        'database_size', pg_database_size(current_database()),
        'uptime', extract(epoch from (now() - pg_postmaster_start_time()))
    ) AS details;

-- Log successful initialization
DO $
BEGIN
    RAISE NOTICE 'GoDash database initialization completed successfully at %', NOW();
    RAISE NOTICE 'Database: %, User: %', current_database(), current_user;
END $;
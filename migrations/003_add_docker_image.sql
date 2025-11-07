-- Migration: Add Docker image fields to submissions table
-- Created: 2025-11-07
-- Description: Add fields to support Kaniko-based Docker image builds

BEGIN;

-- Add Docker image related columns
ALTER TABLE submissions 
ADD COLUMN IF NOT EXISTS docker_image_url VARCHAR(512),
ADD COLUMN IF NOT EXISTS build_job_name VARCHAR(128),
ADD COLUMN IF NOT EXISTS build_pod_name VARCHAR(128);

-- Add index on build_job_name for faster lookups
CREATE INDEX IF NOT EXISTS idx_submissions_build_job_name 
ON submissions(build_job_name) 
WHERE build_job_name IS NOT NULL;

-- Add index on docker_image_url to quickly find built submissions
CREATE INDEX IF NOT EXISTS idx_submissions_docker_image_url 
ON submissions(docker_image_url) 
WHERE docker_image_url IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN submissions.docker_image_url IS 'Docker registry image URL (e.g., registry.io/agent-123:v1)';
COMMENT ON COLUMN submissions.build_job_name IS 'Kubernetes Job name used to build this submission';
COMMENT ON COLUMN submissions.build_pod_name IS 'Kubernetes Pod name that executed the build';

COMMIT;

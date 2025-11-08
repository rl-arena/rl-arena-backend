-- 005_add_security_scan.sql
-- Trivy 보안 스캔 결과 저장을 위한 테이블 추가

ALTER TABLE submissions
ADD COLUMN security_scan_status VARCHAR(50) DEFAULT 'pending',
ADD COLUMN security_scan_result JSONB,
ADD COLUMN security_scan_summary TEXT,
ADD COLUMN vulnerability_count_critical INTEGER DEFAULT 0,
ADD COLUMN vulnerability_count_high INTEGER DEFAULT 0,
ADD COLUMN vulnerability_count_medium INTEGER DEFAULT 0,
ADD COLUMN vulnerability_count_low INTEGER DEFAULT 0,
ADD COLUMN scanned_at TIMESTAMP;

CREATE INDEX idx_submissions_security_scan_status ON submissions(security_scan_status);
CREATE INDEX idx_submissions_vulnerability_critical ON submissions(vulnerability_count_critical);

COMMENT ON COLUMN submissions.security_scan_status IS 'pending, scanning, completed, failed';
COMMENT ON COLUMN submissions.security_scan_result IS 'Full Trivy scan JSON result';
COMMENT ON COLUMN submissions.security_scan_summary IS 'Human-readable summary of vulnerabilities';

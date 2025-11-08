# Database Reset PowerShell Script
# Run this to completely reset the RL Arena database

$ErrorActionPreference = "Stop"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "RL Arena Database Reset Script" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "WARNING: This will DELETE ALL DATA in the database!" -ForegroundColor Red
Write-Host ""

$confirm = Read-Host "Type 'RESET' to confirm"

if ($confirm -ne "RESET") {
    Write-Host "Reset cancelled." -ForegroundColor Yellow
    exit 0
}

Write-Host ""
Write-Host "Connecting to PostgreSQL..." -ForegroundColor Green

# Database connection details
$PGHOST = "127.0.0.1"
$PGPORT = "5433"
$PGUSER = "postgres"
$PGPASSWORD = "postgres"
$PGDATABASE = "rl_arena"

# Set environment variables for PostgreSQL
$env:PGPASSWORD = $PGPASSWORD

# Read SQL file
$sqlFile = Join-Path $PSScriptRoot "..\scripts\reset_database.sql"

if (-not (Test-Path $sqlFile)) {
    Write-Host "ERROR: SQL file not found at $sqlFile" -ForegroundColor Red
    exit 1
}

$sqlContent = Get-Content $sqlFile -Raw

Write-Host "Executing reset script..." -ForegroundColor Green

# Try using docker exec if PostgreSQL is running in Docker
try {
    docker exec -i rl-arena-postgres psql -U $PGUSER -d $PGDATABASE -c "$sqlContent"
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "Database reset complete!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "1. Restart the backend server" -ForegroundColor White
    Write-Host "2. Register new users in the web interface" -ForegroundColor White
    Write-Host "3. Create agents and submit code" -ForegroundColor White
    Write-Host ""
} catch {
    Write-Host "Docker method failed. Trying direct psql connection..." -ForegroundColor Yellow
    
    # Check if psql is available
    $psqlPath = Get-Command psql -ErrorAction SilentlyContinue
    
    if (-not $psqlPath) {
        Write-Host ""
        Write-Host "========================================" -ForegroundColor Yellow
        Write-Host "PostgreSQL client not found in PATH" -ForegroundColor Yellow
        Write-Host "========================================" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Please execute the SQL manually:" -ForegroundColor Cyan
        Write-Host "1. Open DBeaver or pgAdmin" -ForegroundColor White
        Write-Host "2. Connect to: $PGHOST`:$PGPORT/$PGDATABASE" -ForegroundColor White
        Write-Host "3. Execute the SQL file: $sqlFile" -ForegroundColor White
        Write-Host ""
        
        # Open the SQL file in default editor
        Write-Host "Opening SQL file for you..." -ForegroundColor Green
        Start-Process $sqlFile
        exit 1
    }
    
    # Try direct psql connection
    $sqlContent | psql -h $PGHOST -p $PGPORT -U $PGUSER -d $PGDATABASE
    
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "Database reset complete!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "1. Restart the backend server" -ForegroundColor White
    Write-Host "2. Register new users in the web interface" -ForegroundColor White
    Write-Host "3. Create agents and submit code" -ForegroundColor White
    Write-Host ""
}

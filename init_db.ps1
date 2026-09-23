# Database initialization script
# Function: Execute order_init.sql in model/mysql directory to initialize the order_db database
# Note: This operation will DROP and recreate order_db, confirmation is required before execution

$OriginalDir = Get-Location
$SqlDir = Join-Path $PSScriptRoot "model\mysql"
$SqlFile = Join-Path $SqlDir "id_generator_init.sql"

try {
    Write-Host "========================================" -ForegroundColor Yellow
    Write-Host "       Database Initialization Script" -ForegroundColor Yellow
    Write-Host "========================================" -ForegroundColor Yellow
    Write-Host "SQL directory: $SqlDir" -ForegroundColor Cyan
    Write-Host "SQL file: $SqlFile" -ForegroundColor Cyan
    Write-Host ""

    # Secondary confirmation to prevent accidental execution (will drop and recreate id_generator__db)
    $confirm = Read-Host "This will drop and recreate the id_generator_db database! Type y to confirm, any other key to cancel"
    if ($confirm -ne 'y') {
        Write-Host "Execution cancelled." -ForegroundColor Red
        exit 0
    }

    # Check if the sql file exists
    if (-not (Test-Path $SqlFile)) {
        Write-Host "Error: SQL file not found: $SqlFile" -ForegroundColor Red
        exit 1
    }

    # Switch to model/mysql directory
    Set-Location $SqlDir
    Write-Host "Current directory: $(Get-Location)" -ForegroundColor Cyan

    # Execute the initialization command
    Write-Host "Initializing database..." -ForegroundColor Green
    Get-Content -Encoding UTF8 id_generator_init.sql | mysql -h 127.0.0.1 -P 3306 -u root -proot123456

    if ($LASTEXITCODE -eq 0) {
        Write-Host "Database initialized successfully!" -ForegroundColor Green
    } else {
        Write-Host "Database initialization failed, exit code: $LASTEXITCODE" -ForegroundColor Red
        exit $LASTEXITCODE
    }
} finally {
    # Return to the original directory
    Set-Location $OriginalDir
    Write-Host "Returned to: $(Get-Location)" -ForegroundColor Cyan
}

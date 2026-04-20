Set-Location "D:/moh-projects/moh-clinician-app"
$envFile = ".env"
if (-not (Test-Path $envFile)) { Write-Output "FAIL: .env not found at $(Get-Location)"; exit 1 }

Get-Content $envFile | ForEach-Object {
  $line = $_.Trim()
  if ($line -and -not $line.StartsWith('#') -and $line.Contains('=')) {
    $parts = $line -split '=', 2
    $key = $parts[0].Trim()
    $val = $parts[1].Trim()
    if (($val.StartsWith('"') -and $val.EndsWith('"')) -or ($val.StartsWith("'") -and $val.EndsWith("'"))) {
      $val = $val.Substring(1, $val.Length - 2)
    }
    if ($key -like 'DB_*') { Set-Item -Path ("Env:" + $key) -Value $val }
  }
}

$requiredEnv = 'DB_HOST', 'DB_PORT', 'DB_USER', 'DB_PASSWORD', 'DB_NAME', 'DB_SSLMODE'
$missingEnv = $requiredEnv | Where-Object { -not [Environment]::GetEnvironmentVariable($_) }
if ($missingEnv.Count -gt 0) { Write-Output ("FAIL: Missing DB env vars: " + ($missingEnv -join ', ')); exit 1 }

$psqlCmd = Get-Command psql -ErrorAction SilentlyContinue
if (-not $psqlCmd) { Write-Output "FAIL: psql not found in PATH"; exit 1 }
Write-Output ("psql: " + $psqlCmd.Source)

$env:PGPASSWORD = $env:DB_PASSWORD
$env:PGSSLMODE = $env:DB_SSLMODE

function Run-Query([string]$label, [string]$sql) {
  Write-Output ("--- " + $label + " ---")
  $out = & psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -A -c $sql 2>&1
  $code = $LASTEXITCODE
  if ($code -ne 0) {
    Write-Output ("ERROR(" + $code + "): " + ($out -join "`n"))
  }
  elseif (-not $out -or (($out | Out-String).Trim().Length -eq 0)) {
    Write-Output "(none)"
  }
  else {
    $out | ForEach-Object { $_.ToString().TrimEnd() }
  }
}

$sqlA = "SELECT current_database(), current_user, current_schema();"
$sqlB = "WITH req(tbl) AS (VALUES ('lg'),('facilities'),('departments'),('specialist_titles'),('rights'),('employees'),('employeerights'),('users'),('employee_profile_changes'),('indicators'),('targets'),('leavetypes'),('staffleave'),('department_roles'),('weeklyreport'),('attendance_records'),('surgeries'),('ward_rounds'),('investigations')) SELECT tbl FROM req r WHERE NOT EXISTS (SELECT 1 FROM information_schema.tables t WHERE t.table_schema='clinician_app' AND t.table_name=r.tbl) ORDER BY 1;"
$sqlC = "WITH req(table_name,column_name) AS (VALUES ('weeklyreport','days_worked'),('weeklyreport','submitted_on'),('weeklyreport','submit_status'),('weeklyreport','report_status'),('employees','employee_number'),('employees','date_of_birth'),('employees','phone_number'),('employees','facility'),('employees','department'),('staffleave','leave_type_id'),('staffleave','return_date'),('staffleave','leave_status')) SELECT table_name || '|' || column_name FROM req r WHERE NOT EXISTS (SELECT 1 FROM information_schema.columns c WHERE c.table_schema='clinician_app' AND c.table_name=r.table_name AND c.column_name=r.column_name) ORDER BY 1,2;"
$sqlD = "WITH req(idx) AS (VALUES ('idx_users_username'),('idx_employees_facility'),('idx_employees_department'),('idx_staffleave_employee_id'),('idx_weeklyreport_employee_start'),('idx_weeklyreport_facility_dept_start')) SELECT idx FROM req r WHERE NOT EXISTS (SELECT 1 FROM pg_indexes i WHERE i.schemaname='clinician_app' AND i.indexname=r.idx) ORDER BY 1;"

Run-Query 'A current_database/current_user/current_schema' $sqlA
Run-Query 'B missing required tables (clinician_app)' $sqlB
Run-Query 'C missing required columns (critical tables)' $sqlC
Run-Query 'D missing required indexes (clinician_app)' $sqlD
Write-Output "DONE"

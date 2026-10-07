# scripts/verify_key_status.ps1
# Securely verifies whether a Google Gemini API key is active or revoked without echoing the secret (Decision #18)
param(
    [string]$EnvFile = "D:\Projects\CollabBoard\.env",
    [string]$VarName = "GEMINI_API_KEY"
)

if (-not (Test-Path $EnvFile)) {
    Write-Error "Target file not found: $EnvFile"
    exit 1
}

$line = (Get-Content $EnvFile | Where-Object { $_ -match "^$VarName=" })
if (-not $line) {
    Write-Output "KEY_STATUS: NOT_FOUND_IN_FILE"
    exit 0
}

$key = $line.Substring($VarName.Length + 1).Trim(" '`"")
if ([string]::IsNullOrWhiteSpace($key)) {
    Write-Output "KEY_STATUS: EMPTY_IN_FILE"
    exit 0
}

try {
    $res = Invoke-RestMethod -Uri "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.5-flash-lite:generateContent" `
        -Method Post `
        -Headers @{ "x-goog-api-key" = $key } `
        -Body '{"contents":[{"parts":[{"text":"ping"}]}]}' `
        -ContentType "application/json"
    Write-Output "PROBE_RESULT: ACTIVE (HTTP 200 OK - Key is currently LIVE and functioning)"
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    $stream = $_.Exception.Response.GetResponseStream()
    $err = (New-Object System.IO.StreamReader($stream)).ReadToEnd()
    Write-Output "PROBE_RESULT: REVOKED / INVALID (HTTP $statusCode)"
    Write-Output "ERROR_BODY: $err"
}

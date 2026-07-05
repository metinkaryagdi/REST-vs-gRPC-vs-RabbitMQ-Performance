# PowerShell script to automate performance benchmarks

# Ensure the results directory exists and is clean
if (Test-Path "results") {
    Remove-Item -Recurse -Force results\* -ErrorAction SilentlyContinue
} else {
    New-Item -ItemType Directory -Path "results" | Out-Null
}

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "1. Pre-compiling Server and Client..." -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Compiling servers..." -ForegroundColor Yellow
go build -o server_bin.exe ./server
if ($LASTEXITCODE -ne 0) {
    Write-Error "Server compilation failed."
    exit 1
}

Write-Host "Compiling client..." -ForegroundColor Yellow
go build -o client_bin.exe ./client
if ($LASTEXITCODE -ne 0) {
    Write-Error "Client compilation failed."
    exit 1
}

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "2. Starting RabbitMQ using Docker..." -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
docker-compose up -d

Write-Host "Waiting for RabbitMQ to become healthy on port 5672..." -ForegroundColor Yellow
$rabbitmqUp = $false
for ($i = 0; $i -lt 30; $i++) {
    $connection = Test-NetConnection -ComputerName 127.0.0.1 -Port 5672 -WarningAction SilentlyContinue -InformationLevel Quiet
    if ($connection) {
        Write-Host "RabbitMQ is up and running!" -ForegroundColor Green
        $rabbitmqUp = $true
        break
    }
    Start-Sleep -Seconds 1
}

if (-not $rabbitmqUp) {
    Write-Error "RabbitMQ failed to start in 30 seconds. Aborting benchmarks."
    docker-compose down
    exit 1
}

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "3. Launching REST & gRPC Servers..." -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# Start servers using pre-compiled binary
$serverProc = Start-Process -FilePath ".\server_bin.exe" -NoNewWindow -PassThru
Start-Sleep -Seconds 2 # Allow server to bind ports instantly

Write-Host "Checking if servers started successfully..." -ForegroundColor Yellow
$restCheck = Test-NetConnection -ComputerName 127.0.0.1 -Port 8080 -WarningAction SilentlyContinue -InformationLevel Quiet
$grpcCheck = Test-NetConnection -ComputerName 127.0.0.1 -Port 50051 -WarningAction SilentlyContinue -InformationLevel Quiet

if (-not ($restCheck -and $grpcCheck)) {
    Write-Error "Servers failed to start correctly. Check port 8080 (REST) and 50051 (gRPC)."
    Stop-Process -Id $serverProc.Id -Force
    docker-compose down
    exit 1
}
Write-Host "Servers are up!" -ForegroundColor Green

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "4. Running Benchmark Matrix..." -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

$concurrencies = @(1, 10, 50, 100, 200, 500)
$modes = @("rest", "grpc", "rabbitmq_confirm", "rabbitmq_async", "rabbitmq_e2e")
$payloads = @("small", "large")
$outputFile = "results/benchmark_results.json"

foreach ($payload in $payloads) {
    foreach ($mode in $modes) {
        foreach ($c in $concurrencies) {
            # Scale request count to keep execution fast but statistically valid
            $reqCount = 20000
            if ($payload -eq "large") {
                if ($c -eq 1) { $reqCount = 1000 }
                elseif ($c -eq 10) { $reqCount = 2000 }
                else { $reqCount = 5000 }
            } else {
                if ($c -eq 1) { $reqCount = 5000 }
                elseif ($c -eq 10) { $reqCount = 10000 }
            }
            
            # Execute pre-compiled client
            .\client_bin.exe --mode $mode --concurrency $c --requests $reqCount --payload-size $payload --output $outputFile
            
            # Sleep slightly to let socket connections settle and RabbitMQ garbage collect
            Start-Sleep -Seconds 1
        }
    }
}

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "5. Cleaning up resources..." -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

Write-Host "Stopping Go servers..." -ForegroundColor Yellow
Stop-Process -Id $serverProc.Id -Force

Write-Host "Stopping RabbitMQ container..." -ForegroundColor Yellow
docker-compose down

Write-Host "Removing temporary binaries..." -ForegroundColor Yellow
Remove-Item -Force .\server_bin.exe
Remove-Item -Force .\client_bin.exe

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "6. Generating Performance Charts..." -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
python scripts/plot_results.py

Write-Host "Benchmarks Completed Successfully!" -ForegroundColor Green

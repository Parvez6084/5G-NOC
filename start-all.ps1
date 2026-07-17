# start-all.ps1 — launches all 5G-NOC services, each in its own terminal window

$root = "D:\Project\netpulse-ai"

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root\services\telemetry-collector'; go run main.go"
Start-Sleep -Seconds 2

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root\services\telemetry-simulator'; go run main.go"
Start-Sleep -Seconds 1

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root\services\api-gateway'; go run main.go"
Start-Sleep -Seconds 1

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root\services\anomaly-engine'; go run main.go"
Start-Sleep -Seconds 1

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root\services\anomaly-ml'; venv\Scripts\activate; uvicorn main:app --port 8083 --reload"
Start-Sleep -Seconds 1

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root\dashboard-react'; npm run dev"

Write-Host "All 5G-NOC services starting in separate windows..."

curl "http://localhost:${PORT:-3000}/targets" -X PUT -d 'name=c' -d 'target=http://localhost:5555'


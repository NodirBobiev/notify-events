# Notification Service


## How to run:


To start the server, run the following command in the repository root:
```
$ go run .
```


Make HTTP POST request in a separate terminal:
```
$ curl -X POST -H "Content-Type: application/json" -d '{"orderType": "Purchase","sessionId": "29827525-06c9-4b1e-9d9b-7c4584e82f56","card": "4433**1409","eventDate": "2023-01-04 13:44:52.835626 +00:00","websiteUrl": "https://amazon.com"}' http://localhost:8080

```


## How to run tests:

```
go test .
```


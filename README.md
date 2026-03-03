# go-icq [![Godoc Reference](https://pkg.go.dev/badge/github.com/pchchv/go-icq)](https://pkg.go.dev/github.com/pchchv/go-icq) [![Go Report Card](https://goreportcard.com/badge/github.com/pchchv/go-icq)](https://goreportcard.com/report/github.com/pchchv/go-icq)

**Go ICQ** is an self-hostable instant messaging server compatible for AIM and ICQ clients.

## Features:

**AIM**

- [x] Windows AIM Clients
- [x] Away Messages
- [x] Buddy Icons (v4.x, v5.x)
- [x] Buddy List
- [x] Chat Rooms
- [x] Public & Private Chat Exchanges
- [x] Instant Messaging
- [x] User Profiles
- [x] Privacy (allow or block specific users)
- [x] Warning
- [x] User Directory Search
- [x] TOC Protocol Clients: Quick Buddy, gaim, TiK
- [x] File Sharing
    - LAN Only: Direct Connect, Get File
    - Lan/Internet

**ICQ**

- [x] Windows ICQ Clients: 2000b (more to come soon)
- [x] Instant Messaging
- [x] Profiles
- [x] User Search
- [x] Presence Statuses
- [x] Offline Messaging

## Management API

The Management API provides functionality for administering the server.
The following shows you how to run these commands via the command line.

### Windows PowerShell

> Run these commands from **PowerShell**.

#### List Users

```powershell
Invoke-WebRequest -Uri http://localhost:8080/user -Method Get
```

#### Create Users

```powershell
Invoke-WebRequest -Uri http://localhost:8080/user `
  -Body '{"screen_name":"MyScreenName", "password":"thepassword"}' `
  -Method Post `
  -ContentType "application/json"
```

#### Delete Users

```powershell
Invoke-WebRequest -Uri http://localhost:8080/user `
  -Body '{"screen_name": "user123"}' `
  -Method Delete `
  -ContentType "application/json"
```

#### Change Password

```powershell
Invoke-WebRequest -Uri http://localhost:8080/user/password `
  -Body '{"screen_name":"MyScreenName", "password":"thenewpassword"}' `
  -Method Put `
  -ContentType "application/json"
```

#### List Active Sessions

This request lists sessions for all logged in users.

```powershell
Invoke-WebRequest -Uri http://localhost:8080/session -Method Get
```

#### Create Public Chat Room

```powershell
Invoke-WebRequest -Uri http://localhost:8080/chat/room/public `
  -Body '{"name":"Office Hijinks"}' `
  -Method Post `
  -ContentType "application/json"
```

#### List Public Chat Rooms

```powershell
Invoke-WebRequest -Uri http://localhost:8080/chat/room/public -Method Get
```

### Linux / macOS

#### List Users

```shell
curl http://localhost:8080/user
```

#### Create Users

##### AIM

```shell
curl -d'{"screen_name":"MyScreenName", "password":"thepassword"}' http://localhost:8080/user
```

##### ICQ

```shell
curl -d'{"screen_name":"100003", "password":"thepassw"}' http://localhost:8080/user
```

#### Delete Users

```shell
curl -X DELETE -d '{"screen_name": "user123"}' http://localhost:8080/user
```

#### Change Password

```shell
curl -X PUT -d'{"screen_name":"MyScreenName", "password":"thenewpassword"}' http://localhost:8080/user/password
```

#### List Active Sessions

This request lists sessions for all logged in users.

```shell
curl http://localhost:8080/session
```

#### Create Public Chat Room

```shell
curl -d'{"name":"Office Hijinks"}' http://localhost:8080/chat/room/public
```

#### List Public Chat Rooms

```shell
curl http://localhost:8080/chat/room/public
```
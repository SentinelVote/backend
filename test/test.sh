#!/bin/sh
curl -X POST http://localhost:8080/login \
-H "Content-Type: application/json" \
-d '{
      "email": "does_not_exist@example.com",
      "password": "invalid"
    }'

curl -X POST http://localhost:8080/login \
-H "Content-Type: application/json" \
-d '{
      "email": "admin@sentinelvote.tech",
      "password": "Password1!"
    }'
    
curl -X POST http://localhost:8080/login \
-H "Content-Type: application/json" \
-d '{
      "email": "user1@sentinelvote.tech",
      "password": "Password1!"
    }'

curl -X POST http://localhost:8080/login \
-H "Content-Type: application/json" \
-d '{
      "email": "user2@sentinelvote.tech",
      "password": "Password1!"
    }'

curl -X POST http://localhost:8080/login \
-H "Content-Type: application/json" \
-d '{
      "email": "user3@sentinelvote.tech",
      "password": "password"
    }'
    
curl -X POST http://localhost:8080/login \
-H "Content-Type: application/json" \
-d '{
      "email": "user3@sentinelvote.tech",
      "password": "invalid"
    }'
    
curl -X POST http://localhost:8080/login \
-H "Content-Type: application/json" \
-d '{
      "email": "user3@sentinelvote.tech"
    }'
    
curl -X POST http://localhost:8080/login \
-H "Content-Type: application/json" \
-d '{
      "password": "invalid"
    }'
    
curl -X POST http://localhost:8080/login \
-H "Content-Type: application/json" \
-d '{}'


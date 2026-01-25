
- [ ] change the uuid in redis methods to string

- [x] complete the monitor module
    - [x] write SQL migrations
    - [x] write SQL queries
    - [x] write repo layer
    - [x] write svc layer
    - [x] write handler layer
    - [x] connect routes

- [ ] complete the user module , it should be very lean 
    - [x] write SQL migrations and queries
    - [x] write repo layer
    - [x] write svc layer 
    - [x] write handler 
    - [x] connect routes
    - [ ] write just imp middleware and simple auth (access token logic)

- [ ] connect all in container and router

- [ ] write down main func

- [ ] write gracefull shutdown

- [ ] now test all - locally

- [ ] write down docker file, test it in docker compose (3 instances, redis, postgres)

- [ ] write HLD, full Architure, optimizations, challenges, capabilities in a readme.md file

- [ ] build a frontend from alkush 

- [ ] add it in resume


Quick rule of thumb for sqlc

:exec → INSERT/UPDATE/DELETE without RETURNING

:one → exactly one row returned

:many → multiple rows

:execrows → want rowsAffected

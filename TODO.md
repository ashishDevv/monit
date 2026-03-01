# Todos
---
- [x] change the uuid in redis methods to string

- [x] complete the monitor module
    - [x] write SQL migrations
    - [x] write SQL queries
    - [x] write repo layer
    - [x] write svc layer
    - [x] write handler layer
    - [x] connect routes

- [x] complete the user module , it should be very lean 
    - [x] write SQL migrations and queries
    - [x] write repo layer
    - [x] write svc layer 
    - [x] write handler 
    - [x] connect routes
    - [x] complete monitor inicident methods as well
    - [x] write just imp middleware and simple auth (access token logic)

- [x] connect all in container and router

- [x] write down main func

- [x] write gracefull shutdown

- [x] Do proper error handling and logging now

- [x] now test all - locally

- [x] write down docker file, test it in docker compose (3 instances, redis, postgres)

- [x] write HLD, full Architure, optimizations, challenges, capabilities in a readme.md file

- [x] build a frontend from alkush 

- [x] add it in resume


## Future Work

- [ ] Add unit and integration Tests
- [ ] Implement Observablity with OpenTelementry
  - [ ] logging
  - [ ] Metrics
  - [ ] Traces
 



Quick rule of thumb for sqlc

:exec → INSERT/UPDATE/DELETE without RETURNING

:one → exactly one row returned

:many → multiple rows

:execrows → want rowsAffected

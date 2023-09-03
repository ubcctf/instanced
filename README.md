# ubcctf/instanced

Manages challenge instances on-demand.

`instanced` runs in the cluster and exposes an HTTP API which is used to request instances.

- GET `/instances` - get list of active instances
- POST `/instances?chal=$CHALLNAME&team=$ID` - provision an instance for specific challenge and team
- DELETE `/instances?id=$ID` - delete challenge with id

Authenticate with `Bearer token`

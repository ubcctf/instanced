# ubcctf/instanced

currently Jank As Hell.

Manages challenge instances on-demand.

`instanced` runs in the cluster and exposes an HTTP API which is used to request instances.
Challenge templates are added in the form of CRDs. Example format is in this repository.
`instanced` must be restarted every time new CRDs are applied.

Instances created are kept track of in a local sqlite database. The instancer periodically scans the database for expired instances and deletes them.


- GET `/instances` - get list of active instances
- GET `/challenges?team=$ID` - get list of available challenges and instance states for specific team
- POST `/instances?chal=$CHALLNAME&team=$ID` - provision an instance for specific challenge and team
- DELETE `/instances?id=$ID` - delete challenge with id


```mermaid
sequenceDiagram
    actor User
    participant CTFd
    participant instanced
    participant k as kube-apiserver
    User->>CTFd: Load Instances Page
    CTFd->>instanced: GET /challenges?team=ID
    instanced->>CTFd: [{expiry, name, url}, ...]
    User->>CTFd: Restart Instance
    CTFd->>instanced: POST /instances?chal=CHALLNAME&team=ID
    instanced--)k: Create Objects
    instanced->>CTFd: URL of new instance
    CTFd->>User: Instance Created
    Note over instanced: Instance expires
    instanced--)k: Destroy Objects
```
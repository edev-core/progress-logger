# progress-logger
An API to log progress in a event using git.

## API

- GET `/{event-id}/commits`: Gives a list of all commits
- GET `/{event-id}/projects`: Gives a list of all registered projects
- POST `/{event-id}/join`: Registers a project for an event
- POST `/create`: Creates a new event

## Storage
Storage is using bolt, a KVDB, here are the buckets
- Projects : `projects` (Event) -> (List(Project))
- Commits : `commits` (Event) -> (List(Commits))

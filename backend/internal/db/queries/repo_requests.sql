-- name: CreateRepoRequest :one
insert into repo_requests (name, description, template_slug, private, with_lambda, requested_by, requested_by_email)
values ($1, $2, $3, $4, $5, $6, $7)
returning *;

-- name: GetRepoRequest :one
select * from repo_requests
where id = $1;

-- name: ListRepoRequestsByUser :many
select * from repo_requests
where requested_by = $1
order by created_at desc
limit 100;

-- name: MarkRepoRequestRunning :exec
update repo_requests
set status = 'running', error = null, updated_at = now()
where id = $1;

-- name: MarkRepoRequestSucceeded :exec
update repo_requests
set status = 'succeeded', repo_url = $2, error = null, updated_at = now()
where id = $1;

-- name: MarkRepoRequestFailed :exec
update repo_requests
set status = 'failed', error = $2, updated_at = now()
where id = $1;

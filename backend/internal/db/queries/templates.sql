-- name: ListTemplates :many
select * from templates
order by name;

-- name: GetTemplate :one
select * from templates
where slug = $1;

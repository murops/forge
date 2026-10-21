-- +goose Up
create table templates (
    slug            text primary key,
    name            text not null,
    description     text not null default '',
    source_repo     text not null,
    supports_lambda boolean not null default false,
    created_at      timestamptz not null default now()
);

insert into templates (slug, name, description, source_repo, supports_lambda) values
    ('api-python', 'API Python', 'FastAPI service with tests and Dockerfile', 'murops-forge/template-api-python', false),
    ('api-go', 'API Go', 'Go HTTP service with tests and Dockerfile', 'murops-forge/template-api-go', false),
    ('lambda-python', 'Lambda Python', 'Python function deployed to AWS Lambda as a container image', 'murops-forge/template-lambda-python', true);

create type request_status as enum ('pending', 'running', 'succeeded', 'failed');

create table repo_requests (
    id                 uuid primary key default gen_random_uuid(),
    name               text not null,
    description        text not null default '',
    template_slug      text not null references templates (slug),
    private            boolean not null default true,
    with_lambda        boolean not null default false,
    status             request_status not null default 'pending',
    repo_url           text,
    error              text,
    requested_by       text not null,
    requested_by_email text not null default '',
    created_at         timestamptz not null default now(),
    updated_at         timestamptz not null default now()
);

-- a failed request frees the name so it can be retried
create unique index repo_requests_name_active on repo_requests (name) where status <> 'failed';
create index repo_requests_requested_by on repo_requests (requested_by, created_at desc);

-- +goose Down
drop table repo_requests;
drop type request_status;
drop table templates;

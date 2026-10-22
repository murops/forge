-- +goose Up
alter table templates add column required_checks text[] not null default '{}';

-- job names of the templates' CI workflow
update templates set required_checks = '{test,trivy,sonar}';

-- +goose Down
alter table templates drop column required_checks;

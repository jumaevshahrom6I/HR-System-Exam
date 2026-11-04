Begin;

create table if not exists roles(
    id serial PRIMARY KEY,
    code varchar(20) not NULL UNIQUE,
    name varchar(50) NOT NULL,
    created_at TIMESTAMPTZ not NULL DEFAULT now()
);

create table if not exists users(
    id serial PRIMARY KEY,
    role_id INT not null REFERENCES roles(id),
    email varchar(100) not null unique check(email = lower(email)),
    password_hash VARCHAR(255) not null,
    is_active bool not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null DEFAULT now()
);


Create index if not exists idx_users_role_id on users(role_id);
insert into roles(code, name)
values ('hr', 'HR'), ('employee', 'Сотрудник')
on conflict (code) do nothing;

Commit;